package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
)

const transactionIdSQLPg13 string = `
SELECT
    pg_catalog.pg_current_xact_id(),
	next_multixact_id
FROM pg_catalog.pg_control_checkpoint()
`

const transactionIdSQLDefault string = `
SELECT
    pg_catalog.txid_current(),
	next_multixact_id
FROM pg_catalog.pg_control_checkpoint()
`

const xminHorizonSQL string = `
SELECT
COALESCE((
	SELECT
		CASE WHEN COALESCE(pg_catalog.age(backend_xid), 0) > COALESCE(pg_catalog.age(backend_xmin), 0)
			THEN backend_xid
			ELSE backend_xmin
		END
	FROM pg_catalog.pg_stat_activity
	WHERE backend_xmin IS NOT NULL OR backend_xid IS NOT NULL
	ORDER BY greatest(pg_catalog.age(backend_xmin), pg_catalog.age(backend_xid)) DESC
	LIMIT 1
), '0'::xid) as backend,
COALESCE((
	SELECT
		xmin
	FROM pg_catalog.pg_replication_slots
	WHERE xmin IS NOT NULL
	ORDER BY pg_catalog.age(xmin) DESC
	LIMIT 1
), '0'::xid) as replication_slot_xmin,
COALESCE((
	SELECT
		catalog_xmin
	FROM pg_catalog.pg_replication_slots
	WHERE catalog_xmin IS NOT NULL
	ORDER BY pg_catalog.age(catalog_xmin) DESC
	LIMIT 1
), '0'::xid) as replication_slot_catalog_xmin,
COALESCE((
	SELECT
		transaction AS xmin
	FROM pg_catalog.pg_prepared_xacts
	ORDER BY pg_catalog.age(transaction) DESC
	LIMIT 1
), '0'::xid) as prepare_xact,
COALESCE((
	SELECT
		backend_xmin
	FROM (
		SELECT
			backend_xmin
		FROM %s
			UNION ALL
		SELECT
			feedback_xmin::text::xid as backend_xmin
		FROM %s
		WHERE feedback_xmin IS NOT NULL
	) _(backend_xmin)
	ORDER BY pg_catalog.age(backend_xmin) DESC
	LIMIT 1
), '0'::xid) as standby
`

const pgStatStatementsInfoSQL string = `
SELECT
	dealloc,
	stats_reset
FROM %s;
`

const ioStatisticSQLPg16 string = `
SELECT backend_type,
	   object,
	   context,
	   coalesce(reads, 0),
	   coalesce(read_time, 0),
	   coalesce(writes, 0),
	   coalesce(write_time, 0),
	   coalesce(writebacks, 0),
	   coalesce(writeback_time, 0),
	   coalesce(extends, 0),
	   coalesce(extend_time, 0),
	   coalesce(hits, 0),
	   coalesce(evictions, 0),
	   coalesce(reuses, 0),
	   coalesce(fsyncs, 0),
	   coalesce(fsync_time, 0)
  FROM pg_stat_io
`

func GetServerStats(ctx context.Context, c *Collection, db *sql.DB, ps state.PersistedState, ts state.TransientState) (state.PersistedState, state.TransientState, error) {
	var stats state.PostgresServerStats
	var transactionIdSQL string

	err := getPgStatStatementsInfo(ctx, db, &ps.PgStatStatementsStats)
	if err != nil {
		return ps, ts, err
	}

	// Only collect transaction ID or xmin horizon related stats with non-replicas
	if isReplica, err := getIsReplica(ctx, db); err == nil && !isReplica {
		// Query xmin horizon before querying the current transaction ID
		// as the backend_xmin from pg_stat_activity can point to the "next" transaction ID.
		var sourceStatReplicationTable string

		if c.HelperExists("get_stat_replication", nil) {
			c.Logger.PrintVerbose("Found pganalyze.get_stat_replication() stats helper")
			sourceStatReplicationTable = "pganalyze.get_stat_replication()"
		} else {
			sourceStatReplicationTable = "pg_catalog.pg_stat_replication"
		}

		var auroraStatsRel string
		if c.PostgresVersion.IsAwsAurora {
			auroraStatsRel = "aurora_replica_status()"
		} else {
			auroraStatsRel = "(VALUES(NULL)) AS _(feedback_xmin)"
		}
		err = db.QueryRowContext(ctx, QueryMarkerSQL+fmt.Sprintf(xminHorizonSQL, sourceStatReplicationTable, auroraStatsRel)).Scan(
			&stats.XminHorizonBackend, &stats.XminHorizonReplicationSlot, &stats.XminHorizonReplicationSlotCatalog,
			&stats.XminHorizonPreparedXact, &stats.XminHorizonStandby,
		)
		if err != nil {
			ts.ServerStats = stats
			return ps, ts, err
		}

		if ts.Version.Numeric >= state.PostgresVersion13 {
			transactionIdSQL = transactionIdSQLPg13
		} else {
			transactionIdSQL = transactionIdSQLDefault
		}

		err = db.QueryRowContext(ctx, QueryMarkerSQL+transactionIdSQL).Scan(
			&stats.CurrentXactId, &stats.NextMultiXactId,
		)
		if err != nil {
			ts.ServerStats = stats
			return ps, ts, err
		}
	}

	ts.ServerStats = stats
	return ps, ts, nil
}

func getPgStatStatementsInfo(ctx context.Context, db *sql.DB, stats *state.PgStatStatementsStats) error {
	var extSchema string
	var foundExtMinorVersion int16
	// pg_stat_statements_info view was introduced in pg_stat_statements 1.9+ (Postgres 14+)
	const supportedExtMinorVersion = 9
	var pgStatStatementsInfoView string

	err := db.QueryRowContext(ctx, QueryMarkerSQL+statementExtensionVersionSQL).Scan(&extSchema, &foundExtMinorVersion)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if foundExtMinorVersion < supportedExtMinorVersion {
		return nil
	}

	pgStatStatementsInfoView = extSchema + ".pg_stat_statements_info"
	err = db.QueryRowContext(ctx, QueryMarkerSQL+fmt.Sprintf(pgStatStatementsInfoSQL, pgStatStatementsInfoView)).Scan(
		&stats.Dealloc, &stats.Reset,
	)
	if err != nil {
		var e *pq.Error
		if errors.As(err, &e) && e.Code == "55000" { // object_not_in_prerequisite_state
			// pg_stat_statements is not loaded via shared_preload_libraries; ignore the error
			// to allow collection to continue (this is handled the same way for other
			// pg_stat_statements queries in input/full.go)
			return nil
		}
	}
	return err
}

func GetPgStatIo(ctx context.Context, c *Collection, db *sql.DB) (stats state.PostgresServerIoStatsMap, err error) {
	stats = make(state.PostgresServerIoStatsMap)
	var rows *sql.Rows
	if c.PostgresVersion.Numeric < state.PostgresVersion16 {
		return
	}
	rows, err = db.Query(QueryMarkerSQL + ioStatisticSQLPg16)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var k state.PostgresServerIoStatsKey
		var s state.PostgresServerIoStats
		err = rows.Scan(&k.BackendType, &k.IoObject, &k.IoContext,
			&s.Reads, &s.ReadTime, &s.Writes, &s.WriteTime,
			&s.Writebacks, &s.WritebackTime, &s.Extends,
			&s.ExtendTime, &s.Hits, &s.Evictions,
			&s.Reuses, &s.Fsyncs, &s.FsyncTime,
		)
		if err != nil {
			return
		}
		stats[k] = s
	}
	err = rows.Err()
	return
}
