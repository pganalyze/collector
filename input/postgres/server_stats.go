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

func GetServerStats(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, systemType string) (state.PostgresServerStats, error) {
	var stats state.PostgresServerStats
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
			return stats, err
		}
	}

	return stats, nil
}
