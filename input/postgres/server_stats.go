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

const xminHorizonSQL string = `
WITH slots AS (
	SELECT
		xmin,
		catalog_xmin
	FROM pg_replication_slots
	WHERE xmin IS NOT NULL
	ORDER BY age(xmin) DESC
	LIMIT 1
)
SELECT
(
	SELECT
		backend_xmin
	FROM pg_stat_activity
	WHERE backend_xmin IS NOT NULL OR backend_xid IS NOT NULL
	ORDER BY greatest(age(backend_xmin), age(backend_xid)) DESC
	LIMIT 1
) as backend,
(
	SELECT xmin FROM slots
) as replication_slot_xmin,
(
	SELECT catalog_xmin FROM slots
) as replication_slot_catalog_xmin,
(
	SELECT
		transaction AS xmin
	FROM pg_prepared_xacts
	ORDER BY age(transaction) DESC
	LIMIT 1
) as prepare_xact,
(
	SELECT
		backend_xmin
	FROM pg_stat_replication
	WHERE backend_xmin IS NOT NULL
	ORDER BY age(backend_xmin) DESC
	LIMIT 1
) as standby
`

func GetServerStats(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, systemType string) (state.PostgresServerStats, error) {
	var stats state.PostgresServerStats
	var transactionIdSQL string

	// Only collect transaction ID or xmin horizon related stats with non-replicas
	if isReplica, err := getIsReplica(db); err == nil && !isReplica {
		// Query xmin horizon before querying the current transaction ID
		// as the backend_xmin from pg_stat_activity can point to the "next" transaction ID.
		err = db.QueryRow(QueryMarkerSQL+xminHorizonSQL).Scan(
			&stats.XminHorizonBackend, &stats.XminHorizonReplicationSlot, &stats.XminHorizonReplicationSlotCatalog,
			&stats.XminHorizonPreparedXact, &stats.XminHorizonStandby,
		)
		if err != nil {
			return stats, err
		}

		if postgresVersion.Numeric >= state.PostgresVersion13 {
			transactionIdSQL = transactionIdSQLPg13
		} else {
			transactionIdSQL = transactionIdSQLDefault
		}

		err = db.QueryRow(QueryMarkerSQL+transactionIdSQL).Scan(
			&stats.CurrentXactId, &stats.NextMultiXactId,
		)
		if err != nil {
			return stats, err
		}
	}

	return stats, nil
}
