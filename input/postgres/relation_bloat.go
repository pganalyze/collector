package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// Estimation queries by PostgreSQL Experts, licensed under the BSD-3-Clause license
// https://github.com/pgexperts/pgx_scripts

const tableBloatSQL string = `
WITH constants AS (
		-- define some constants for sizes of things
		-- for reference down the query and easy maintenance
		SELECT pg_catalog.current_setting('block_size')::numeric AS bs,
		       23 AS hdr,
					 8 AS ma
),
columns AS (
	SELECT n.nspname AS table_schema,
				 c.relname AS table_name,
				 a.attname AS column_name
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON (c.oid = a.attrelid)
		JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	 WHERE c.relkind IN ('r', 'm')
	       AND a.attnum > 0
				 AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
				 AND NOT a.attisdropped
),
no_stats AS (
		-- screen out table who have attributes
		-- which dont have stats, such as JSON
		SELECT table_schema,
		       table_name,
				   n_live_tup::numeric AS est_rows,
				   pg_catalog.pg_table_size(relid)::numeric AS table_size
		  FROM columns
	 	  JOIN pg_catalog.pg_stat_user_tables psut ON (table_schema = psut.schemaname AND table_name = psut.relname)
			LEFT OUTER JOIN %s ON (table_schema = pg_stats.schemaname AND table_name = pg_stats.tablename AND column_name = attname)
	   WHERE attname IS NULL
		 GROUP BY table_schema, table_name, relid, n_live_tup
),
null_headers AS (
		-- calculate null header sizes
		-- omitting tables which dont have complete stats
		-- and attributes which aren't visible
		SELECT
				hdr + 1 + (pg_catalog.sum(CASE WHEN null_frac <> 0 THEN 1 else 0 END) / 8) AS nullhdr,
				pg_catalog.sum((1 - null_frac) * avg_width) AS datawidth,
				pg_catalog.max(null_frac) AS maxfracsum,
				schemaname,
				tablename,
				hdr, ma, bs
		FROM %s
	 CROSS JOIN constants
	  LEFT OUTER JOIN no_stats ON (schemaname = no_stats.table_schema AND tablename = no_stats.table_name)
	 WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
				 AND no_stats.table_name IS NULL
				 AND EXISTS(SELECT 1 FROM columns
								     WHERE schemaname = columns.table_schema
										       AND tablename = columns.table_name)
	 GROUP BY schemaname, tablename, hdr, ma, bs
),
data_headers AS (
		-- estimate header and row size
		SELECT ma, bs, hdr, schemaname, tablename,
				   (datawidth + (hdr + ma - (CASE WHEN hdr %% ma = 0 THEN ma ELSE hdr %% ma END)))::numeric AS datahdr,
				   (maxfracsum * (nullhdr + ma - (CASE WHEN nullhdr %% ma = 0 THEN ma ELSE nullhdr %% ma END))) AS nullhdr2
		  FROM null_headers
),
table_estimates AS (
		-- make estimates of how large the table should be
		-- based on row and page size
		SELECT schemaname, tablename, bs,
		       reltuples::numeric AS est_rows,
					 relpages * bs AS table_bytes,
					 pg_catalog.ceil((reltuples * (datahdr + nullhdr2 + 4 + ma -
													 (CASE WHEN datahdr %% ma = 0 THEN ma ELSE datahdr %% ma END)
													 ) / (bs - 20))) * bs AS expected_bytes,
				   reltoastrelid
	  	FROM data_headers
			     JOIN pg_catalog.pg_class c ON (tablename = c.relname)
				   JOIN pg_catalog.pg_namespace n ON (c.relnamespace = n.oid AND schemaname = n.nspname)
	   WHERE c.relkind = 'r'
),
estimates_with_toast AS (
		-- add in estimated TOAST table sizes
		-- estimate based on 4 toast tuples per page because we dont have
		-- anything better.  also append the no_data tables
		SELECT schemaname,
					 tablename,
				   est_rows,
				   table_bytes + (COALESCE(toast.relpages, 0) * bs) AS table_bytes,
				   expected_bytes + (pg_catalog.ceil(COALESCE(toast.reltuples, 0) / 4) * bs) AS expected_bytes
	  	FROM table_estimates
		       LEFT OUTER JOIN pg_class AS toast ON (table_estimates.reltoastrelid = toast.oid AND toast.relkind = 't')
)
SELECT schemaname, tablename,
			 CASE WHEN table_bytes > 0
						THEN table_bytes::NUMERIC
						ELSE NULL::NUMERIC END,
			 CASE WHEN expected_bytes > 0 AND table_bytes > 0
						AND expected_bytes <= table_bytes
						THEN (table_bytes - expected_bytes)::NUMERIC
						ELSE 0::NUMERIC END
	FROM estimates_with_toast
`

const indexBloatSQL string = `
WITH btree_index_atts AS (
	SELECT nspname,
				 ic.relname as index_name,
			 	 ic.reltuples,
			 	 ic.relpages,
				 indrelid,
				 indexrelid,
				 ic.relam,
				 tc.relname AS tablename,
				 regexp_split_to_table(indkey::text, ' ')::smallint AS attnum,
				 indexrelid AS index_oid
		FROM pg_catalog.pg_index i
		     JOIN pg_catalog.pg_class AS ic ON (i.indexrelid = ic.oid)
				 JOIN pg_catalog.pg_class AS tc ON (i.indrelid = tc.oid)
				 JOIN pg_catalog.pg_namespace n ON (n.oid = ic.relnamespace)
				 JOIN pg_catalog.pg_am a ON (ic.relam = a.oid)
	 WHERE a.amname = 'btree' AND ic.relpages > 0
				 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
),
index_item_sizes AS (
	SELECT ia.nspname,
				 ia.index_name,
				 ia.reltuples,
				 ia.relpages,
				 ia.relam,
				 indrelid AS table_oid,
				 index_oid,
				 pg_catalog.current_setting('block_size')::numeric AS bs,
				 8 AS maxalign,
				 24 AS pagehdr,
				 CASE WHEN pg_catalog.max(COALESCE(s.null_frac, 0)) = 0 THEN 2 ELSE 6 END AS index_tuple_hdr,
				 pg_catalog.sum((1 - coalesce(s.null_frac, 0)) * COALESCE(s.avg_width, 1024)) AS nulldatawidth
		FROM pg_catalog.pg_attribute a
		JOIN btree_index_atts AS ia ON (a.attrelid = ia.indexrelid AND a.attnum = ia.attnum)
		JOIN %s s ON (s.schemaname = ia.nspname
				          AND ((s.tablename = ia.tablename AND s.attname = pg_catalog.pg_get_indexdef(a.attrelid, a.attnum, TRUE))
				               OR (s.tablename = ia.index_name AND s.attname = a.attname)))
		WHERE a.attnum > 0
		GROUP BY 1, 2, 3, 4, 5, 6, 7, 8, 9
),
index_aligned_est AS (
		SELECT maxalign, bs, nspname, index_name, reltuples,
				relpages, relam, table_oid, index_oid,
				COALESCE(
						pg_catalog.ceil(
								reltuples * ( 6
										+ maxalign
										- CASE
												WHEN index_tuple_hdr %% maxalign = 0 THEN maxalign
												ELSE index_tuple_hdr %% maxalign
											END
										+ nulldatawidth
										+ maxalign
										- CASE /* Add padding to the data to align on MAXALIGN */
												WHEN nulldatawidth::integer %% maxalign = 0 THEN maxalign
												ELSE nulldatawidth::integer %% maxalign
											END
								)::numeric
							/ ( bs - pagehdr::NUMERIC )
							+1 )
				 , 0 )
			as expected
		FROM index_item_sizes
)
SELECT nspname,
       index_name,
			 bs * (iae.relpages)::bigint,
			 CASE
			 WHEN iae.relpages <= expected
				 THEN 0
				 ELSE bs * (iae.relpages - expected)::bigint
			 END
	FROM index_aligned_est iae
	JOIN pg_catalog.pg_class c ON (c.oid = iae.table_oid)
`

// TODO: Figure out how to introduce precise bloat queries here, e.g.
// SELECT index_size, index_size * (1.0 - avg_leaf_density / 100.0) FROM pgstatindex('some_index_pkey'::regclass);
// http://blog.ioguix.net/postgresql/2014/03/28/Playing-with-indexes-and-better-bloat-estimate.html

func GetRelationBloat(logger *util.Logger, db *sql.DB, columnStatsSourceTable string) (relBloat []state.PostgresRelationBloat, err error) {
	rows, err := db.Query(QueryMarkerSQL + fmt.Sprintf(tableBloatSQL, columnStatsSourceTable, columnStatsSourceTable))
	if err != nil {
		err = fmt.Errorf("TableBloat/Query: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var row state.PostgresRelationBloat

		err := rows.Scan(&row.SchemaName, &row.RelationName, &row.TotalBytes, &row.BloatBytes)
		if err != nil {
			err = fmt.Errorf("TableBloat/Scan: %s", err)
			return nil, err
		}

		if row.TotalBytes > 0 && row.BloatBytes > 0 {
			relBloat = append(relBloat, row)
		}
	}

	return
}

func GetIndexBloat(logger *util.Logger, db *sql.DB, columnStatsSourceTable string) (indexBloat []state.PostgresIndexBloat, err error) {
	rows, err := db.Query(QueryMarkerSQL + fmt.Sprintf(indexBloatSQL, columnStatsSourceTable))
	if err != nil {
		err = fmt.Errorf("IndexBloat/Query: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var row state.PostgresIndexBloat

		err := rows.Scan(&row.SchemaName, &row.IndexName, &row.TotalBytes, &row.BloatBytes)
		if err != nil {
			err = fmt.Errorf("IndexBloat/Scan: %s", err)
			return nil, err
		}

		if row.TotalBytes > 0 && row.BloatBytes > 0 {
			indexBloat = append(indexBloat, row)
		}
	}

	return
}

func GetBloatStats(logger *util.Logger, db *sql.DB) (report state.PostgresBloatStats, err error) {
	var columnStatsSourceTable string

	if statsHelperExists(db, "get_column_stats") {
		logger.PrintVerbose("Found pganalyze.get_column_stats() stats helper")
		columnStatsSourceTable = "(SELECT * FROM pganalyze.get_column_stats()) pg_stats"
	} else {
		if !connectedAsSuperUser(db) && !connectedAsMonitoringRole(db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser to run the bloat report.")
		}
		columnStatsSourceTable = "pg_catalog.pg_stats"
	}

	report.Relations, err = GetRelationBloat(logger, db, columnStatsSourceTable)
	if err != nil {
		return
	}

	report.Indices, err = GetIndexBloat(logger, db, columnStatsSourceTable)
	if err != nil {
		return
	}

	report.DatabaseName, err = CurrentDatabaseName(db)
	if err != nil {
		return
	}

	return
}
