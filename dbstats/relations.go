package dbstats

import (
  "database/sql"
  null "gopkg.in/guregu/null.v2"
)

type Relation struct {
  Oid int64 `json:"oid"`
  SchemaName string `json:"schema_name"`
  TableName string `json:"table_name"`
  RelationType string `json:"relation_type"`
  Stats RelationStats `json:"stats"`
  Columns []Column `json:"columns"`
}

type RelationStats struct {
  SizeBytes null.Int `json:"size_bytes"`
  WastedBytes null.Int `json:"wasted_bytes"`
  SeqScan null.Int `json:"seq_scan"`                     // Number of sequential scans initiated on this table
  SeqTupRead null.Int `json:"seq_tup_read"`              // Number of live rows fetched by sequential scans
  IdxScan null.Int `json:"idx_scan"`                     // Number of index scans initiated on this table
  IdxTupFetch null.Int `json:"idx_tup_fetch"`            // Number of live rows fetched by index scans
  NTupIns null.Int `json:"n_tup_ins"`                    // Number of rows inserted
  NTupUpd null.Int `json:"n_tup_upd"`                    // Number of rows updated
  NTupDel null.Int `json:"n_tup_del"`                    // Number of rows deleted
  NTupHotUpd null.Int `json:"n_tup_hot_upd"`             // Number of rows HOT updated (i.e., with no separate index update required)
  NLiveTup null.Int `json:"n_live_tup"`                  // Estimated number of live rows
  NDeadTup null.Int `json:"n_dead_tup"`                  // Estimated number of dead rows
  NModSinceAnalyze null.Int `json:"n_mod_since_analyze"` // Estimated number of rows modified since this table was last analyzed
  LastVacuum Timestamp `json:"last_vacuum"`              // Last time at which this table was manually vacuumed (not counting VACUUM FULL)
  LastAutovacuum Timestamp `json:"last_autovacuum"`      // Last time at which this table was vacuumed by the autovacuum daemon
  LastAnalyze Timestamp `json:"last_analyze"`            // Last time at which this table was manually analyzed
  LastAutoanalyze Timestamp `json:"last_autoanalyze"`    // Last time at which this table was analyzed by the autovacuum daemon
  VacuumCount null.Int `json:"vacuum_count"`             // Number of times this table has been manually vacuumed (not counting VACUUM FULL)
  AutovacuumCount null.Int `json:"autovacuum_count"`     // Number of times this table has been vacuumed by the autovacuum daemon
  AnalyzeCount null.Int `json:"analyze_count"`           // Number of times this table has been manually analyzed
  AutoanalyzeCount null.Int `json:"autoanalyze_count"`   // Number of times this table has been analyzed by the autovacuum daemon
  HeapBlksRead null.Int `json:"heap_blks_read"`          // Number of disk blocks read from this table
  HeapBlksHit null.Int `json:"heap_blks_hit"`            // Number of buffer hits in this table
  IdxBlksRead null.Int `json:"idx_blks_read"`            // Number of disk blocks read from all indexes on this table
  IdxBlksHit null.Int `json:"idx_blks_hit"`              // Number of buffer hits in all indexes on this table
  ToastBlksRead null.Int `json:"toast_blks_read"`        // Number of disk blocks read from this table's TOAST table (if any)
  ToastBlksHit null.Int `json:"toast_blks_hit"`          // Number of buffer hits in this table's TOAST table (if any)
  TidxBlksRead null.Int `json:"tidx_blks_read"`          // Number of disk blocks read from this table's TOAST table indexes (if any)
  TidxBlksHit null.Int `json:"tidx_blks_hit"`            // Number of buffer hits in this table's TOAST table indexes (if any)
}

type Column struct {
  Name string `json:"name"`
  DataType string `json:"data_type"`
  DefaultValue string `json:"default_value"`
  NotNull bool `json:"not_null"`
  Position int32 `json:"position"`
}

const relationsSQL string =
`SELECT c.oid,
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
        s.n_mod_since_analyze,
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
        AND n.nspname NOT IN ('pg_catalog', 'information_schema')`

const columnsSQL string =
`SELECT c.oid,
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
       AND n.nspname NOT IN ('pg_catalog', 'information_schema')
       AND a.attnum > 0
       AND NOT a.attisdropped
 ORDER BY a.attnum`

func GetRelations(db *sql.DB) []Relation {
  stmt, err := db.Prepare(relationsSQL)
  checkErr(err)

  defer stmt.Close()

  rows, err := stmt.Query()

  var relations []Relation

  defer rows.Close()
  for rows.Next() {
    var row Relation

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
    checkErr(err)

    relations = append(relations, row)
  }

  return relations
}
