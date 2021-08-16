package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const relationStatsSQLDefaultOptionalFields = "NULL"
const relationStatsSQLpg94OptionalFields = "s.n_mod_since_analyze"

const relationStatsSQL = `
WITH locked_relids AS (SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL)
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
	FROM pg_catalog.pg_class c
	LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
	LEFT JOIN pg_catalog.pg_stat_user_tables s ON (s.relid = c.oid)
	LEFT JOIN pg_catalog.pg_statio_user_tables sio USING (relid)
 WHERE c.oid NOT IN (SELECT relid FROM locked_relids)
       AND c.relkind IN ('r','v','m','p')
			 AND c.relpersistence <> 't'
			 AND c.relname NOT IN ('pg_stat_statements')
			 AND n.nspname NOT IN ('pg_catalog','pg_toast','information_schema')
			 AND ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
`

const indexStatsSQL = `
WITH locked_relids AS (SELECT DISTINCT relation relid FROM pg_catalog.pg_locks WHERE mode = 'AccessExclusiveLock' AND relation IS NOT NULL)
SELECT s.indexrelid,
			 COALESCE(pg_catalog.pg_relation_size(s.indexrelid), 0) AS size_bytes,
			 COALESCE(s.idx_scan, 0),
			 COALESCE(s.idx_tup_read, 0),
			 COALESCE(s.idx_tup_fetch, 0),
			 COALESCE(sio.idx_blks_read, 0),
			 COALESCE(sio.idx_blks_hit, 0)
	FROM pg_catalog.pg_stat_user_indexes s
			 LEFT JOIN pg_catalog.pg_statio_user_indexes sio USING (indexrelid)
 WHERE s.indexrelid NOT IN (SELECT relid FROM locked_relids)
			 AND ($1 = '' OR (s.schemaname || '.' || s.relname) !~* $1)
`

const columnStatsSQL = `
SELECT schemaname, tablename, attname, inherited, null_frac, avg_width, n_distinct, correlation
  FROM %s
 WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
`

const columnStatsHelperSQL = `
SELECT 1 AS enabled
  FROM pg_catalog.pg_proc p
  JOIN pg_catalog.pg_namespace n ON (p.pronamespace = n.oid)
 WHERE n.nspname = 'pganalyze' AND p.proname = 'get_column_stats'
`

func GetRelationStats(db *sql.DB, postgresVersion state.PostgresVersion, ignoreRegexp string) (relStats state.PostgresRelationStatsMap, err error) {
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

	rows, err := stmt.Query(ignoreRegexp)
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

	relStats, err = handleRelationStatsExt(db, relStats, postgresVersion, ignoreRegexp)

	return
}

func GetIndexStats(db *sql.DB, postgresVersion state.PostgresVersion, ignoreRegexp string) (indexStats state.PostgresIndexStatsMap, err error) {
	stmt, err := db.Prepare(QueryMarkerSQL + indexStatsSQL)
	if err != nil {
		err = fmt.Errorf("IndexStats/Prepare: %s", err)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(ignoreRegexp)
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

func GetColumnStats(logger *util.Logger, db *sql.DB, globalCollectionOpts state.CollectionOpts, systemType string) (columnStats state.PostgresColumnStatsMap, err error) {
	var sourceTable string

	helperExists := false
	db.QueryRow(QueryMarkerSQL + columnStatsHelperSQL).Scan(&helperExists)

	if helperExists {
		logger.PrintVerbose("Found pganalyze.get_column_stats() stats helper")
		sourceTable = "pganalyze.get_column_stats()"
	} else {
		if systemType != "heroku" && !connectedAsSuperUser(db, systemType) && globalCollectionOpts.TestRun {
			logger.PrintInfo("Warning: Limited access to table column statistics detected. Please setup" +
				" the monitoring helper function pganalyze.get_column_stats (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)" +
				" or connect as superuser, to get column statistics for all tables.")
		}
		sourceTable = "pg_catalog.pg_stats"
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(columnStatsSQL, sourceTable))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statsMap = make(state.PostgresColumnStatsMap)

	for rows.Next() {
		var s state.PostgresColumnStats

		err := rows.Scan(
			&s.SchemaName, &s.TableName, &s.ColumnName, &s.Inherited, &s.NullFrac, &s.AvgWidth, &s.NDistinct, &s.Correlation)
		if err != nil {
			return nil, err
		}

		key := state.PostgresColumnStatsKey{SchemaName: s.SchemaName, TableName: s.TableName, ColumnName: s.ColumnName}
		statsMap[key] = append(statsMap[key], s)
	}

	return statsMap, nil
}
