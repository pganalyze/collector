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
		SELECT current_setting('block_size')::numeric AS bs, 23 AS hdr, 8 AS ma
),
columns AS (
	SELECT pg_namespace.nspname AS table_schema,
				 pg_class.relname AS table_name,
				 pg_attribute.attname AS column_name
		FROM pg_attribute
		JOIN pg_class ON (pg_class.oid = pg_attribute.attrelid)
		JOIN pg_namespace ON (pg_namespace.oid = pg_class.relnamespace)
	 WHERE pg_class.relkind IN ('r', 'm') AND pg_attribute.attnum > 0
				 AND nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
				 AND NOT attisdropped
),
no_stats AS (
		-- screen out table who have attributes
		-- which dont have stats, such as JSON
		SELECT table_schema, table_name,
				n_live_tup::numeric as est_rows,
				pg_table_size(relid)::numeric as table_size
		FROM columns
				JOIN pg_stat_user_tables as psut
					 ON table_schema = psut.schemaname
					 AND table_name = psut.relname
				LEFT OUTER JOIN %s
				ON table_schema = pg_stats.schemaname
						AND table_name = pg_stats.tablename
						AND column_name = attname
		WHERE attname IS NULL
		GROUP BY table_schema, table_name, relid, n_live_tup
),
null_headers AS (
		-- calculate null header sizes
		-- omitting tables which dont have complete stats
		-- and attributes which aren't visible
		SELECT
				hdr+1+(sum(case when null_frac <> 0 THEN 1 else 0 END)/8) as nullhdr,
				SUM((1-null_frac)*avg_width) as datawidth,
				MAX(null_frac) as maxfracsum,
				schemaname,
				tablename,
				hdr, ma, bs
		FROM %s CROSS JOIN constants
				LEFT OUTER JOIN no_stats
						ON schemaname = no_stats.table_schema
						AND tablename = no_stats.table_name
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
				AND no_stats.table_name IS NULL
				AND EXISTS ( SELECT 1
						FROM columns
								WHERE schemaname = columns.table_schema
										AND tablename = columns.table_name )
		GROUP BY schemaname, tablename, hdr, ma, bs
),
data_headers AS (
		-- estimate header and row size
		SELECT
				ma, bs, hdr, schemaname, tablename,
				(datawidth+(hdr+ma-(case when hdr %% ma=0 THEN ma ELSE hdr %% ma END)))::numeric AS datahdr,
				(maxfracsum*(nullhdr+ma-(case when nullhdr %% ma=0 THEN ma ELSE nullhdr %% ma END))) AS nullhdr2
		FROM null_headers
),
table_estimates AS (
		-- make estimates of how large the table should be
		-- based on row and page size
		SELECT schemaname, tablename, bs,
				reltuples::numeric as est_rows, relpages * bs as table_bytes,
		CEIL((reltuples*
						(datahdr + nullhdr2 + 4 + ma -
								(CASE WHEN datahdr %% ma=0
										THEN ma ELSE datahdr %% ma END)
								)/(bs-20))) * bs AS expected_bytes,
				reltoastrelid
		FROM data_headers
				JOIN pg_class ON tablename = relname
				JOIN pg_namespace ON relnamespace = pg_namespace.oid
						AND schemaname = nspname
		WHERE pg_class.relkind = 'r'
),
estimates_with_toast AS (
		-- add in estimated TOAST table sizes
		-- estimate based on 4 toast tuples per page because we dont have
		-- anything better.  also append the no_data tables
		SELECT schemaname, tablename,
				est_rows,
				table_bytes + ( coalesce(toast.relpages, 0) * bs ) as table_bytes,
				expected_bytes + ( ceil( coalesce(toast.reltuples, 0) / 4 ) * bs ) as expected_bytes
		FROM table_estimates LEFT OUTER JOIN pg_class as toast
				ON table_estimates.reltoastrelid = toast.oid
						AND toast.relkind = 't'
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
				indexclass.relname as index_name,
				indexclass.reltuples,
				indexclass.relpages,
				indrelid, indexrelid,
				indexclass.relam,
				tableclass.relname as tablename,
				regexp_split_to_table(indkey::text, ' ')::smallint AS attnum,
				indexrelid as index_oid
		FROM pg_index
		JOIN pg_class AS indexclass ON pg_index.indexrelid = indexclass.oid
		JOIN pg_class AS tableclass ON pg_index.indrelid = tableclass.oid
		JOIN pg_namespace ON pg_namespace.oid = indexclass.relnamespace
		JOIN pg_am ON indexclass.relam = pg_am.oid
		WHERE pg_am.amname = 'btree' and indexclass.relpages > 0
				 AND nspname NOT IN ('pg_catalog','pg_toast','information_schema')
),
index_item_sizes AS (
	SELECT ind_atts.nspname, ind_atts.index_name,
				 ind_atts.reltuples, ind_atts.relpages, ind_atts.relam,
				 indrelid AS table_oid, index_oid,
				 current_setting('block_size')::numeric AS bs,
				 8 AS maxalign,
				 24 AS pagehdr,
				 CASE WHEN max(coalesce(pg_stats.null_frac,0)) = 0
					THEN 2
					ELSE 6
				 END AS index_tuple_hdr,
				 sum( (1-coalesce(pg_stats.null_frac, 0)) * coalesce(pg_stats.avg_width, 1024) ) AS nulldatawidth
		FROM pg_attribute
		JOIN btree_index_atts AS ind_atts ON pg_attribute.attrelid = ind_atts.indexrelid AND pg_attribute.attnum = ind_atts.attnum
		JOIN %s ON pg_stats.schemaname = ind_atts.nspname
				 AND ( (pg_stats.tablename = ind_atts.tablename AND pg_stats.attname = pg_catalog.pg_get_indexdef(pg_attribute.attrelid, pg_attribute.attnum, TRUE))
				 OR   (pg_stats.tablename = ind_atts.index_name AND pg_stats.attname = pg_attribute.attname))
		WHERE pg_attribute.attnum > 0
		GROUP BY 1, 2, 3, 4, 5, 6, 7, 8, 9
),
index_aligned_est AS (
		SELECT maxalign, bs, nspname, index_name, reltuples,
				relpages, relam, table_oid, index_oid,
				coalesce (
						ceil (
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
SELECT nspname, index_name,
			 bs*(index_aligned_est.relpages)::bigint,
			 CASE
			 WHEN index_aligned_est.relpages <= expected
				 THEN 0
				 ELSE bs*(index_aligned_est.relpages-expected)::bigint
			 END
	FROM index_aligned_est
	JOIN pg_class ON (pg_class.oid = index_aligned_est.table_oid)
`

const columnStatsHelperSQL string = `
SELECT 1 AS enabled
	FROM pg_proc
	JOIN pg_namespace ON (pronamespace = pg_namespace.oid)
 WHERE nspname = 'pganalyze' AND proname = 'get_column_stats'
`

func columnStatsHelperExists(db *sql.DB) bool {
	var enabled bool

	err := db.QueryRow(QueryMarkerSQL + columnStatsHelperSQL).Scan(&enabled)
	if err != nil {
		return false
	}

	return enabled
}

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

	if columnStatsHelperExists(db) {
		logger.PrintVerbose("Found pganalyze.get_column_stats() stats helper")
		columnStatsSourceTable = "(SELECT * FROM pganalyze.get_column_stats()) pg_stats"
	} else {
		if !connectedAsSuperUser(db) {
			logger.PrintInfo("Warning: You are not connecting as superuser. Please setup" +
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser to run the bloat report.")
		}
		columnStatsSourceTable = "pg_stats"
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
