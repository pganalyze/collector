package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"gopkg.in/guregu/null.v2"
)

const tableBloatSQL string = `
WITH constants AS (
	SELECT current_setting('block_size')::numeric AS bs, 23 AS hdr, 8 AS ma
),
no_stats AS (
	SELECT table_schema, table_name
	 FROM information_schema.columns
	 LEFT OUTER JOIN pg_stats ON table_schema = schemaname
															 AND table_name = tablename
															 AND column_name = attname
	WHERE attname IS NULL
				AND table_schema NOT IN ('pg_catalog','pg_toast','information_schema')
	GROUP BY table_schema, table_name
),
null_headers AS (
	SELECT hdr+1+(sum(case when null_frac <> 0 THEN 1 else 0 END)/8) as nullhdr,
				 SUM((1-null_frac)*avg_width) as datawidth,
				 MAX(null_frac) as maxfracsum,
				 schemaname,
				 tablename,
				 hdr, ma, bs
		FROM pg_stats CROSS JOIN constants
		LEFT OUTER JOIN no_stats ON schemaname = no_stats.table_schema
																AND tablename = no_stats.table_name
	 WHERE schemaname NOT IN ('pg_catalog','pg_toast','information_schema')
				 AND no_stats.table_name IS NULL
				 AND EXISTS (SELECT 1
											 FROM information_schema.columns
											WHERE schemaname = columns.table_schema
														AND tablename = columns.table_name)
	 GROUP BY schemaname, tablename, hdr, ma, bs
),
data_headers AS (
	SELECT ma, bs, hdr, schemaname, tablename,
				 (datawidth+(hdr+ma-(case when hdr % ma=0 THEN ma ELSE hdr % ma END)))::numeric AS datahdr,
				 (maxfracsum*(nullhdr+ma-(case when nullhdr % ma=0 THEN ma ELSE nullhdr % ma END))) AS nullhdr2
		FROM null_headers
),
table_estimates AS (
	SELECT pg_class.oid,
				 relpages * bs as table_bytes,
				 CEIL((reltuples*
							(datahdr + nullhdr2 + 4 + ma -
								(CASE WHEN datahdr % ma=0
									THEN ma ELSE datahdr % ma END)
								)/(bs-20))) * bs AS expected_bytes
		FROM data_headers
		JOIN pg_class ON tablename = relname
		JOIN pg_namespace ON relnamespace = pg_namespace.oid
												 AND schemaname = nspname
	 WHERE pg_class.relkind = 'r'
)
SELECT oid,
	CASE WHEN table_bytes > 0
	THEN table_bytes::NUMERIC
	ELSE NULL::NUMERIC END
	AS table_bytes,
	CASE WHEN expected_bytes > 0
	THEN expected_bytes::NUMERIC
	ELSE NULL::NUMERIC END
	AS expected_bytes,
	CASE WHEN expected_bytes > 0 AND table_bytes > 0
	AND expected_bytes <= table_bytes
	THEN (table_bytes - expected_bytes)::NUMERIC
	ELSE 0::NUMERIC END AS wasted_bytes
FROM table_estimates;
`

const indexBloatSQL string = `
WITH btree_index_atts AS (
	SELECT nspname, relname, reltuples, relpages, indrelid, relam,
				 regexp_split_to_table(indkey::text, ' ')::smallint AS attnum,
				 indexrelid as index_oid
		FROM pg_index
		JOIN pg_class ON pg_class.oid=pg_index.indexrelid
		JOIN pg_namespace ON pg_namespace.oid = pg_class.relnamespace
		JOIN pg_am ON pg_class.relam = pg_am.oid
	 WHERE pg_am.amname = 'btree' AND pg_class.relpages > 0
				 AND nspname NOT IN ('pg_catalog','pg_toast','information_schema')
),
index_item_sizes AS (
	SELECT i.nspname,
				 i.relname,
				 i.reltuples,
				 i.relpages,
				 i.relam,
				 (quote_ident(s.schemaname) || '.' || quote_ident(s.tablename))::regclass AS starelid,
				 a.attrelid AS table_oid,
				 index_oid,
				 current_setting('block_size')::numeric AS bs,
				 8 AS maxalign,
				 24 AS pagehdr,
				 /* per tuple header: add index_attribute_bm if some cols are null-able */
				 CASE WHEN max(coalesce(s.null_frac, 0)) = 0
						 THEN 2
						 ELSE 6
				 END AS index_tuple_hdr,
				 /* data len: we remove null values save space using it fractionnal part from stats */
				 sum( (1 - coalesce(s.null_frac, 0)) * coalesce(s.avg_width, 1024) ) AS nulldatawidth
		FROM pg_attribute a
		JOIN pg_stats s ON (quote_ident(s.schemaname) || '.' || quote_ident(s.tablename))::regclass = a.attrelid AND s.attname = a.attname
		JOIN btree_index_atts i ON i.indrelid = a.attrelid AND a.attnum = i.attnum
	 WHERE a.attnum > 0
	 GROUP BY 1, 2, 3, 4, 5, 6, 7, 8, 9
),
index_aligned AS (
	SELECT maxalign, bs, nspname, relname AS index_name, reltuples,
				 relpages, relam, table_oid, index_oid,
				 ( 6
					 + maxalign
					 /* Add padding to the index tuple header to align on MAXALIGN */
					 - CASE
							 WHEN index_tuple_hdr % maxalign = 0 THEN maxalign
							 ELSE index_tuple_hdr % maxalign
						 END
					 + nulldatawidth
					 + maxalign
					 /* Add padding to the data to align on MAXALIGN */
					 - CASE
							 WHEN nulldatawidth::integer % maxalign = 0 THEN maxalign
							 ELSE nulldatawidth::integer % maxalign
						 END
				)::numeric AS nulldatahdrwidth, pagehdr
	 FROM index_item_sizes
),
otta_calc AS (
	SELECT bs, nspname, table_oid, index_oid, index_name, relpages,
				 coalesce(
						ceil(reltuples * nulldatahdrwidth)::numeric / bs
						- pagehdr::numeric
						/* btree and hash have a metadata reserved block */
						+ CASE WHEN am.amname IN ('hash', 'btree') THEN 1 ELSE 0 END,
						0
				 ) AS otta
	FROM index_aligned
	LEFT JOIN pg_am am ON index_aligned.relam = am.oid
)
SELECT sub.index_oid,
       pg_catalog.pg_relation_size(s.relid) AS size_bytes,
	CASE
		WHEN sub.relpages <= otta THEN 0
		ELSE bs * (sub.relpages - otta)::bigint
	END AS wasted_bytes
FROM otta_calc AS sub
		 JOIN pg_class AS c ON c.oid = sub.table_oid
		 JOIN pg_stat_user_indexes AS stat ON sub.index_oid = stat.indexrelid
`

func GetRelationBloat(db *sql.DB, postgresVersion state.PostgresVersion) (relBloat state.PostgresRelationBloatMap, err error) {
	stmt, err := db.Prepare(QueryMarkerSQL + tableBloatSQL)
	if err != nil {
		err = fmt.Errorf("TableBloat/Prepare: %s", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		err = fmt.Errorf("TableBloat/Query: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var oid state.Oid
		var sizeBytes null.Int
		var expectedBytes null.Int
		var wastedBytes null.Int

		err := rows.Scan(&oid, &sizeBytes, &expectedBytes, &wastedBytes)
		if err != nil {
			err = fmt.Errorf("TableBloat/Scan: %s", err)
			return nil, err
		}

		if sizeBytes.Valid && wastedBytes.Valid {
			relBloat[oid] = state.PostgresRelationBloat{WastedBytes: wastedBytes.Int64, SizeBytes: sizeBytes.Int64}
		}
	}

	return
}

func GetIndexBloat(db *sql.DB, postgresVersion state.PostgresVersion) (indexBloat state.PostgresIndexBloatMap, err error) {
	stmt, err := db.Prepare(QueryMarkerSQL + indexBloatSQL)
	if err != nil {
		err = fmt.Errorf("IndexBloat/Prepare: %s", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		err = fmt.Errorf("IndexBloat/Query: %s", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var oid state.Oid
		var sizeBytes null.Int
		var wastedBytes null.Int

		err := rows.Scan(&oid, &sizeBytes, &wastedBytes)
		if err != nil {
			err = fmt.Errorf("IndexBloat/Scan: %s", err)
			return nil, err
		}

		if sizeBytes.Valid && wastedBytes.Valid {
			indexBloat[oid] = state.PostgresIndexBloat{WastedBytes: wastedBytes.Int64, SizeBytes: sizeBytes.Int64}
		}
	}

	return
}
