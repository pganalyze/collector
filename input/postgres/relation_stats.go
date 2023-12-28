package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
)

const relationStatsSQLInsertsSinceVacuumFieldPg13 string = "s.n_ins_since_vacuum"
const relationStatsSQLInsertsSinceVacuumFieldDefault string = "0"

const relationStatsSQL = `
WITH locked_relids AS (
	SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL AND locktype = 'relation'
),
locked_relids_with_parents AS (
	SELECT DISTINCT inhparent relid FROM pg_catalog.pg_inherits WHERE inhrelid IN (SELECT relid FROM locked_relids)
	UNION SELECT relid FROM locked_relids
)
SELECT c.oid,
			 COALESCE(pg_catalog.pg_table_size(c.oid), 0) +
			 COALESCE((SELECT pg_catalog.sum(pg_catalog.pg_table_size(inhrelid))
			    FROM pg_catalog.pg_inherits
				 WHERE inhparent = c.oid), 0) AS size_bytes,
			 CASE c.reltoastrelid WHEN NULL THEN 0 ELSE COALESCE(pg_catalog.pg_total_relation_size(c.reltoastrelid), 0) END AS toast_bytes,
			 COALESCE(s.seq_scan, 0),
			 COALESCE(s.seq_tup_read, 0),
			 COALESCE(s.idx_scan, 0),
			 COALESCE(s.idx_tup_fetch, 0),
			 COALESCE(s.n_tup_ins, 0),
			 COALESCE(s.n_tup_upd, 0),
			 COALESCE(s.n_tup_del, 0),
			 COALESCE(s.n_tup_hot_upd, 0),
			 COALESCE(s.n_live_tup, 0),
			 COALESCE(s.n_dead_tup, 0),
			 COALESCE(s.n_mod_since_analyze, 0),
			 COALESCE(%s, 0),
			 s.last_vacuum,
			 s.last_autovacuum,
			 s.last_analyze,
			 s.last_autoanalyze,
			 COALESCE(s.vacuum_count, 0),
			 COALESCE(s.autovacuum_count, 0),
			 COALESCE(s.analyze_count, 0),
			 COALESCE(s.autoanalyze_count, 0),
			 COALESCE(sio.heap_blks_read, 0),
			 COALESCE(sio.heap_blks_hit, 0),
			 COALESCE(sio.idx_blks_read, 0),
			 COALESCE(sio.idx_blks_hit, 0),
			 COALESCE(sio.toast_blks_read, 0),
			 COALESCE(sio.toast_blks_hit, 0),
			 COALESCE(sio.tidx_blks_read, 0),
			 COALESCE(sio.tidx_blks_hit, 0),
			 CASE WHEN c.relfrozenxid <> '0' THEN pg_catalog.age(c.relfrozenxid) ELSE 0 END AS relation_xid_age,
			 CASE WHEN c.relminmxid <> '0' THEN pg_catalog.mxid_age(c.relminmxid) ELSE 0 END AS relation_mxid_age,
			 c.relpages,
			 c.reltuples,
			 c.relallvisible,
			 false AS exclusively_locked,
			 COALESCE(toast.reltuples, -1) AS toast_reltuples,
			 COALESCE(toast.relpages, 0) AS toast_relpages
	FROM pg_catalog.pg_class c
	LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	LEFT JOIN pg_catalog.pg_stat_user_tables s ON (s.relid = c.oid)
	LEFT JOIN pg_catalog.pg_statio_user_tables sio USING (relid)
	LEFT JOIN pg_class toast ON (c.reltoastrelid = toast.oid AND toast.relkind = 't')
 WHERE c.oid NOT IN (SELECT relid FROM locked_relids_with_parents)
       AND c.relkind IN ('r','v','m','p')
			 AND c.relpersistence <> 't'
			 AND c.oid NOT IN (SELECT pd.objid FROM pg_catalog.pg_depend pd WHERE pd.deptype = 'e' AND pd.classid = 'pg_catalog.pg_class'::regclass)
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
			 AND ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
 UNION ALL
SELECT relid,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   NULL,
	   NULL,
	   NULL,
	   NULL,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   true AS exclusively_locked,
	   0,
	   0
  FROM locked_relids_with_parents
`

const indexStatsSQL = `
WITH locked_relids AS (SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL AND locktype = 'relation')
SELECT s.indexrelid,
			 COALESCE(pg_catalog.pg_relation_size(s.indexrelid), 0) AS size_bytes,
			 COALESCE(s.idx_scan, 0),
			 COALESCE(s.idx_tup_read, 0),
			 COALESCE(s.idx_tup_fetch, 0),
			 COALESCE(sio.idx_blks_read, 0),
			 COALESCE(sio.idx_blks_hit, 0),
			 false AS exclusively_locked
	FROM pg_catalog.pg_stat_user_indexes s
			 LEFT JOIN pg_catalog.pg_statio_user_indexes sio USING (indexrelid)
 WHERE s.indexrelid NOT IN (SELECT relid FROM locked_relids)
			 AND ($1 = '' OR (s.schemaname || '.' || s.relname) !~* $1)
UNION ALL
SELECT relid,
	   0,
	   0,
	   0,
	   0,
	   0,
	   0,
	   true AS exclusively_locked
  FROM locked_relids
`

func GetRelationStats(ctx context.Context, db *sql.DB, postgresVersion state.PostgresVersion, server *state.Server) (relStats state.PostgresRelationStatsMap, err error) {
	var insertsSinceVacuumField string

	if postgresVersion.Numeric >= state.PostgresVersion13 {
		insertsSinceVacuumField = relationStatsSQLInsertsSinceVacuumFieldPg13
	} else {
		insertsSinceVacuumField = relationStatsSQLInsertsSinceVacuumFieldDefault
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(relationStatsSQL, insertsSinceVacuumField))
	if err != nil {
		err = fmt.Errorf("RelationStats/Prepare: %s", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, server.Config.IgnoreSchemaRegexp)
	if err != nil {
		err = fmt.Errorf("RelationStats/Query: %s", err)
		return
	}
	defer rows.Close()

	relStats = make(state.PostgresRelationStatsMap)
	for rows.Next() {
		var oid state.Oid
		var stats state.PostgresRelationStats

		err = rows.Scan(&oid, &stats.SizeBytes, &stats.ToastSizeBytes,
			&stats.SeqScan, &stats.SeqTupRead,
			&stats.IdxScan, &stats.IdxTupFetch, &stats.NTupIns,
			&stats.NTupUpd, &stats.NTupDel, &stats.NTupHotUpd,
			&stats.NLiveTup, &stats.NDeadTup, &stats.NModSinceAnalyze, &stats.NInsSinceVacuum,
			&stats.LastVacuum, &stats.LastAutovacuum, &stats.LastAnalyze,
			&stats.LastAutoanalyze, &stats.VacuumCount, &stats.AutovacuumCount,
			&stats.AnalyzeCount, &stats.AutoanalyzeCount, &stats.HeapBlksRead,
			&stats.HeapBlksHit, &stats.IdxBlksRead, &stats.IdxBlksHit,
			&stats.ToastBlksRead, &stats.ToastBlksHit, &stats.TidxBlksRead,
			&stats.TidxBlksHit, &stats.FrozenXIDAge, &stats.MinMXIDAge,
			&stats.Relpages, &stats.Reltuples, &stats.Relallvisible,
			&stats.ExclusivelyLocked, &stats.ToastReltuples, &stats.ToastRelpages)
		if err != nil {
			err = fmt.Errorf("RelationStats/Scan: %s", err)
			return
		}

		relStats[oid] = stats
	}

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("RelationStats/Rows: %s", err)
		return
	}

	relStats, err = handleRelationStatsAux(ctx, db, relStats, postgresVersion, server)

	return
}

func GetIndexStats(ctx context.Context, db *sql.DB, postgresVersion state.PostgresVersion, server *state.Server) (indexStats state.PostgresIndexStatsMap, err error) {
	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+indexStatsSQL)
	if err != nil {
		err = fmt.Errorf("IndexStats/Prepare: %s", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, server.Config.IgnoreSchemaRegexp)
	if err != nil {
		err = fmt.Errorf("IndexStats/Query: %s", err)
		return
	}
	defer rows.Close()

	indexStats = make(state.PostgresIndexStatsMap)
	for rows.Next() {
		var oid state.Oid
		var stats state.PostgresIndexStats

		err = rows.Scan(&oid, &stats.SizeBytes, &stats.IdxScan, &stats.IdxTupRead,
			&stats.IdxTupFetch, &stats.IdxBlksRead, &stats.IdxBlksHit,
			&stats.ExclusivelyLocked)
		if err != nil {
			err = fmt.Errorf("IndexStats/Scan: %s", err)
			return
		}

		indexStats[oid] = stats
	}

	if err = rows.Err(); err != nil {
		err = fmt.Errorf("IndexStats/Rows: %s", err)
		return
	}

	indexStats, err = handleIndexStatsAux(ctx, db, indexStats, postgresVersion, server)

	return
}
