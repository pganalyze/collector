package dbstats

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/snapshot"

	null "gopkg.in/guregu/null.v2"
)

const relationsSQLDefaultOptionalFields = "NULL"
const relationsSQLpg94OptionalFields = "s.n_mod_since_analyze"

const relationsSQL string = `SELECT c.oid,
				n.nspname AS schema_name,
				c.relname AS table_name,
				c.relkind AS relation_type,
				pg_catalog.pg_table_size(c.oid) AS size_bytes,
				s.seq_scan,
				s.seq_tup_read,
				s.idx_scan,
				s.idx_tup_fetch,
				s.n_tup_ins,
				s.n_tup_upd,
				s.n_tup_del,
				s.n_tup_hot_upd,
				s.n_live_tup,
				s.n_dead_tup,
				%s,
				s.last_vacuum,
				s.last_autovacuum,
				s.last_analyze,
				s.last_autoanalyze,
				s.vacuum_count,
				s.autovacuum_count,
				s.analyze_count,
				s.autoanalyze_count,
				sio.heap_blks_read,
				sio.heap_blks_hit,
				sio.idx_blks_read,
				sio.idx_blks_hit,
				sio.toast_blks_read,
				sio.toast_blks_hit,
				sio.tidx_blks_read,
				sio.tidx_blks_hit
	 FROM pg_catalog.pg_class c
	 LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	 LEFT JOIN pg_catalog.pg_stat_user_tables s ON (s.relid = c.oid)
	 LEFT JOIN pg_catalog.pg_statio_user_tables sio ON (sio.relid = c.oid)
	WHERE c.relkind IN ('r','v','m')
				AND c.relpersistence <> 't'
				AND c.relname NOT IN ('pg_stat_statements')
				AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

const columnsSQL string = `SELECT c.oid,
				a.attname AS name,
				pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
	 (SELECT pg_catalog.pg_get_expr(d.adbin, d.adrelid)
		FROM pg_catalog.pg_attrdef d
		WHERE d.adrelid = a.attrelid
			AND d.adnum = a.attnum
			AND a.atthasdef) AS default_value,
				a.attnotnull AS not_null,
				a.attnum AS position
 FROM pg_catalog.pg_class c
 LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
 LEFT JOIN pg_catalog.pg_attribute a ON c.oid = a.attrelid
 WHERE c.relkind IN ('r','v','m')
			 AND c.relpersistence <> 't'
			 AND c.relname NOT IN ('pg_stat_statements')
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
			 AND a.attnum > 0
			 AND NOT a.attisdropped
 ORDER BY a.attnum`

const indicesSQL string = `
SELECT c.oid,
			 c2.oid AS index_oid,
			 i.indkey::text AS columns,
			 c2.relname AS name,
			 pg_catalog.pg_relation_size(c2.oid) AS size_bytes,
			 i.indisprimary AS is_primary,
			 i.indisunique AS is_unique,
			 i.indisvalid AS is_valid,
			 pg_catalog.pg_get_indexdef(i.indexrelid, 0, TRUE) AS index_def,
			 pg_catalog.pg_get_constraintdef(con.oid, TRUE) AS constraint_def,
			 s.idx_scan, s.idx_tup_read, s.idx_tup_fetch,
			 sio.idx_blks_read, sio.idx_blks_hit
	FROM pg_catalog.pg_class c
	JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	JOIN pg_catalog.pg_index i ON (c.oid = i.indrelid)
	JOIN pg_catalog.pg_class c2 ON (i.indexrelid = c2.oid)
	LEFT JOIN pg_catalog.pg_constraint con ON (conrelid = i.indrelid
																						 AND conindid = i.indexrelid
																						 AND contype IN ('p', 'u', 'x'))
	LEFT JOIN pg_stat_user_indexes s ON (s.indexrelid = c2.oid)
	LEFT JOIN pg_statio_user_indexes sio ON (sio.indexrelid = c2.oid)
 WHERE c.relkind IN ('r','v','m')
			 AND c.relpersistence <> 't'
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

// FIXME: This misses check constraints and others
const constraintsSQL string = `
SELECT c.oid,
			 conname AS name,
			 pg_catalog.pg_get_constraintdef(r.oid, TRUE) AS constraint_def,
			 r.conkey AS columns,
			 n2.nspname AS foreign_schema,
			 c2.relname AS foreign_table,
			 r.confkey AS foreign_columns
	FROM pg_catalog.pg_class c
			 LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			 LEFT JOIN pg_catalog.pg_constraint r ON r.conrelid = c.oid
			 LEFT JOIN pg_catalog.pg_class c2 ON r.confrelid = c2.oid
			 LEFT JOIN pg_catalog.pg_namespace n2 ON n2.oid = c2.relnamespace
WHERE r.contype = 'f'
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

const viewDefinitionSQL string = `
SELECT c.oid,
			 pg_catalog.pg_get_viewdef(c.oid) AS view_definition
	FROM pg_catalog.pg_class c
	LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	WHERE c.relkind IN ('v','m')
			 AND c.relpersistence <> 't'
			 AND c.relname NOT IN ('pg_stat_statements')
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')`

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
SELECT sub.table_oid,
			 sub.index_oid,
	CASE
		WHEN sub.relpages <= otta THEN 0
		ELSE bs * (sub.relpages - otta)::bigint
	END AS wasted_bytes
FROM otta_calc AS sub
		 JOIN pg_class AS c ON c.oid = sub.table_oid
		 JOIN pg_stat_user_indexes AS stat ON sub.index_oid = stat.indexrelid
`

func GetRelations(db *sql.DB, postgresVersion snapshot.PostgresVersion, collectBloat bool) ([]*snapshot.Relation, error) {
	var optionalFields string

	relations := make(map[int64]snapshot.Relation, 0)

	if postgresVersion.Numeric >= snapshot.PostgresVersion94 {
		optionalFields = relationsSQLpg94OptionalFields
	} else {
		optionalFields = relationsSQLDefaultOptionalFields
	}

	// Relations
	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(relationsSQL, optionalFields))
	if err != nil {
		err = fmt.Errorf("Relations/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		err = fmt.Errorf("Relations/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row snapshot.Relation

		row.Stats = &snapshot.Relation_Stats{}

		err := rows.Scan(&row.Oid, &row.SchemaName, &row.TableName, &row.RelationType,
			&row.Stats.SizeBytes, &row.Stats.SeqScan, &row.Stats.SeqTupRead,
			&row.Stats.IdxScan, &row.Stats.IdxTupFetch, &row.Stats.NTupIns,
			&row.Stats.NTupUpd, &row.Stats.NTupDel, &row.Stats.NTupHotUpd,
			&row.Stats.NLiveTup, &row.Stats.NDeadTup, &row.Stats.NModSinceAnalyze,
			&row.Stats.LastVacuum, &row.Stats.LastAutovacuum, &row.Stats.LastAnalyze,
			&row.Stats.LastAutoanalyze, &row.Stats.VacuumCount, &row.Stats.AutovacuumCount,
			&row.Stats.AnalyzeCount, &row.Stats.AutoanalyzeCount, &row.Stats.HeapBlksRead,
			&row.Stats.HeapBlksHit, &row.Stats.IdxBlksRead, &row.Stats.IdxBlksHit,
			&row.Stats.ToastBlksRead, &row.Stats.ToastBlksHit, &row.Stats.TidxBlksRead,
			&row.Stats.TidxBlksHit)
		if err != nil {
			err = fmt.Errorf("Relations/Scan: %s", err)
			return nil, err
		}

		relations[row.Oid] = row
	}

	// Columns
	stmt, err = db.Prepare(QueryMarkerSQL + columnsSQL)
	if err != nil {
		err = fmt.Errorf("Columns/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err = stmt.Query()
	if err != nil {
		err = fmt.Errorf("Columns/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row snapshot.Relation_Column

		err := rows.Scan(&row.RelationOid, &row.Name, &row.DataType, &row.DefaultValue,
			&row.NotNull, &row.Position)
		if err != nil {
			err = fmt.Errorf("Columns/Scan: %s", err)
			return nil, err
		}

		relation := relations[row.RelationOid]
		relation.Columns = append(relation.Columns, &row)
		relations[row.RelationOid] = relation
	}

	// Indices
	stmt, err = db.Prepare(QueryMarkerSQL + indicesSQL)
	if err != nil {
		err = fmt.Errorf("Indices/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err = stmt.Query()
	if err != nil {
		err = fmt.Errorf("Indices/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row snapshot.Relation_Index

		err := rows.Scan(&row.RelationOid, &row.IndexOid, &row.Columns, &row.Name, &row.SizeBytes,
			&row.IsPrimary, &row.IsUnique, &row.IsValid, &row.IndexDef, &row.ConstraintDef,
			&row.IdxScan, &row.IdxTupRead, &row.IdxTupFetch, &row.IdxBlksRead, &row.IdxBlksHit)
		if err != nil {
			err = fmt.Errorf("Indices/Scan: %s", err)
			return nil, err
		}

		relation := relations[row.RelationOid]
		relation.Indices = append(relation.Indices, &row)
		relations[row.RelationOid] = relation
	}

	// Constraints
	stmt, err = db.Prepare(QueryMarkerSQL + constraintsSQL)
	if err != nil {
		err = fmt.Errorf("Constraints/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err = stmt.Query()
	if err != nil {
		err = fmt.Errorf("Constraints/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var row snapshot.Relation_Constraint

		err := rows.Scan(&row.RelationOid, &row.Name, &row.ConstraintDef, &row.Columns,
			&row.ForeignSchema, &row.ForeignTable, &row.ForeignColumns)
		if err != nil {
			err = fmt.Errorf("Constraints/Scan: %s", err)
			return nil, err
		}

		relation := relations[row.RelationOid]
		relation.Constraints = append(relation.Constraints, &row)
		relations[row.RelationOid] = relation
	}

	// View definitions
	stmt, err = db.Prepare(QueryMarkerSQL + viewDefinitionSQL)
	if err != nil {
		err = fmt.Errorf("Views/Prepare: %s", err)
		return nil, err
	}

	defer stmt.Close()

	rows, err = stmt.Query()
	if err != nil {
		err = fmt.Errorf("Views/Query: %s", err)
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var relationOid int64
		var viewDefinition string

		err := rows.Scan(&relationOid, &viewDefinition)
		if err != nil {
			err = fmt.Errorf("Views/Scan: %s", err)
			return nil, err
		}

		relation := relations[relationOid]
		relation.ViewDefinition = viewDefinition
		relations[relationOid] = relation
	}

	if collectBloat {
		// Table bloat
		stmt, err = db.Prepare(QueryMarkerSQL + tableBloatSQL)
		if err != nil {
			err = fmt.Errorf("TableBloat/Prepare: %s", err)
			return nil, err
		}

		defer stmt.Close()

		rows, err = stmt.Query()
		if err != nil {
			err = fmt.Errorf("TableBloat/Query: %s", err)
			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var relationOid int64
			var tableBytes null.Int
			var expectedBytes null.Int
			var wastedBytes null.Int

			err := rows.Scan(&relationOid, &tableBytes, &expectedBytes, &wastedBytes)
			if err != nil {
				err = fmt.Errorf("TableBloat/Scan: %s", err)
				return nil, err
			}

			if wastedBytes.Valid {
				relation := relations[relationOid]
				relation.Stats.WastedBytes = wastedBytes.Int64
				relations[relationOid] = relation
			}
		}

		// Index bloat
		stmt, err = db.Prepare(QueryMarkerSQL + indexBloatSQL)
		if err != nil {
			err = fmt.Errorf("IndexBloat/Prepare: %s", err)
			return nil, err
		}

		defer stmt.Close()

		rows, err = stmt.Query()
		if err != nil {
			err = fmt.Errorf("IndexBloat/Query: %s", err)
			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var relationOid int64
			var indexOid int64
			var wastedBytes null.Int

			err := rows.Scan(&relationOid, &indexOid, &wastedBytes)
			if err != nil {
				err = fmt.Errorf("IndexBloat/Scan: %s", err)
				return nil, err
			}

			if wastedBytes.Valid {
				relation := relations[relationOid]

				for idx, index := range relation.Indices {
					if index.IndexOid == indexOid {
						index.WastedBytes = wastedBytes.Int64
						relation.Indices[idx] = index
						break
					}
				}

				relations[relationOid] = relation
			}
		}
	}

	// Flip the oid-based map into an array

	v := make([]*snapshot.Relation, 0, len(relations))
	for _, value := range relations {
		v = append(v, &value)
	}

	return v, nil
}
