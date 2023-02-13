package postgres

import (
	"database/sql"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const transactionIdSQLPg13 string = `
SELECT
	pg_current_xact_id(),
	next_multixact_id
FROM pg_catalog.pg_control_checkpoint()
`

const transactionIdSQLDefault string = `
SELECT
	txid_current(),
	next_multixact_id
FROM pg_catalog.pg_control_checkpoint()
`

const ioStatisticSQLPg16 string = `
SELECT backend_type,
	   io_object,
	   io_context,
	   reads,
	   writes,
	   extends,
	   op_bytes,
	   evictions,
	   reuses,
	   fsyncs
  FROM pg_stat_io
`

func GetServerStats(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, systemType string) (state.PostgresServerStats, state.PostgresServerIoStatsMap, error) {
	var stats state.PostgresServerStats
	var ioStats state.PostgresServerIoStatsMap
	var transactionIdSQL string

	// Only collect xact ID related stats with non-replicas
	if isReplica, err := getIsReplica(db); err == nil && !isReplica {
		if postgresVersion.Numeric >= state.PostgresVersion13 {
			transactionIdSQL = transactionIdSQLPg13
		} else {
			transactionIdSQL = transactionIdSQLDefault
		}

		err = db.QueryRow(QueryMarkerSQL+transactionIdSQL).Scan(
			&stats.CurrentXactId, &stats.NextMultiXactId,
		)
		if err != nil {
			return stats, ioStats, err
		}
	}

	// Retrieve I/O statistics if we're on a new enough Postgres
	if postgresVersion.Numeric >= state.PostgresVersion16 {
		rows, err := db.Query(QueryMarkerSQL + ioStatisticSQLPg16)
		if err != nil {
			return stats, ioStats, err
		}
		defer rows.Close()

		ioStats = make(state.PostgresServerIoStatsMap)

		for rows.Next() {
			var k state.PostgresServerIoStatsKey
			var s state.PostgresServerIoStats

			err := rows.Scan(&k.BackendType, &k.IoObject, &k.IoContext,
				&s.Reads, &s.Writes, &s.Extends, &s.OpBytes,
				&s.Evictions, &s.Reuses, &s.Fsyncs,
			)
			if err != nil {
				return stats, ioStats, err
			}

			ioStats[k] = s
		}

		if err = rows.Err(); err != nil {
			return stats, ioStats, err
		}
	}

	return stats, ioStats, nil
}

type PostgresServerIoStatsKey struct {
	BackendType string // a backend type like "autovacuum worker"
	IoObject    string // "relation" or "temp relation"
	IoContext   string // "normal", "vacuum", "bulkread" or "bulkwrite"
}

type PostgresServerIoStats struct {
	Reads     int64
	Writes    int64
	Extends   int64
	OpBytes   int64
	Evictions int64
	Reuses    int64
	Fsyncs    int64
}
