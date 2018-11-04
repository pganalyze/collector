package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
)

const relationStatsSQLDefaultOptionalFields = "NULL"
const relationStatsSQLpg94OptionalFields = "s.n_mod_since_analyze"

const relationStatsSQL = `
SELECT s.relid,
			 COALESCE(pg_catalog.pg_table_size(s.relid), 0) AS size_bytes,
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
			 %s,
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
			 COALESCE(sio.tidx_blks_hit, 0)
	FROM pg_catalog.pg_stat_user_tables s
			 JOIN pg_catalog.pg_class c ON (s.relid = c.oid)
			 LEFT JOIN pg_catalog.pg_statio_user_tables sio USING (relid)
`

const indexStatsSQL = `
SELECT s.indexrelid,
			 COALESCE(pg_catalog.pg_relation_size(s.indexrelid), 0) AS size_bytes,
			 COALESCE(s.idx_scan, 0),
			 COALESCE(s.idx_tup_read, 0),
			 COALESCE(s.idx_tup_fetch, 0),
			 COALESCE(sio.idx_blks_read, 0),
			 COALESCE(sio.idx_blks_hit, 0)
	FROM pg_catalog.pg_stat_user_indexes s
			 LEFT JOIN pg_catalog.pg_statio_user_indexes sio USING (indexrelid)
`

func GetRelationStats(db *sql.DB, postgresVersion state.PostgresVersion) (relStats state.PostgresRelationStatsMap, err error) {
	var optionalFields string

	if postgresVersion.Numeric >= state.PostgresVersion94 {
		optionalFields = relationStatsSQLpg94OptionalFields
	} else {
		optionalFields = relationStatsSQLDefaultOptionalFields
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(relationStatsSQL, optionalFields))
	if err != nil {
		err = fmt.Errorf("RelationStats/Prepare: %s", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
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
			&stats.NLiveTup, &stats.NDeadTup, &stats.NModSinceAnalyze,
			&stats.LastVacuum, &stats.LastAutovacuum, &stats.LastAnalyze,
			&stats.LastAutoanalyze, &stats.VacuumCount, &stats.AutovacuumCount,
			&stats.AnalyzeCount, &stats.AutoanalyzeCount, &stats.HeapBlksRead,
			&stats.HeapBlksHit, &stats.IdxBlksRead, &stats.IdxBlksHit,
			&stats.ToastBlksRead, &stats.ToastBlksHit, &stats.TidxBlksRead,
			&stats.TidxBlksHit)
		if err != nil {
			err = fmt.Errorf("RelationStats/Scan: %s", err)
			return
		}

		relStats[oid] = stats
	}

	return
}

func GetIndexStats(db *sql.DB, postgresVersion state.PostgresVersion) (indexStats state.PostgresIndexStatsMap, err error) {
	stmt, err := db.Prepare(QueryMarkerSQL + indexStatsSQL)
	if err != nil {
		err = fmt.Errorf("IndexStats/Prepare: %s", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query()
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
			&stats.IdxTupFetch, &stats.IdxBlksRead, &stats.IdxBlksHit)
		if err != nil {
			err = fmt.Errorf("IndexStats/Scan: %s", err)
			return
		}

		indexStats[oid] = stats
	}

	return
}
