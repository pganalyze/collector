package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const backendCountsSQLDefaultOptionalFields = "CASE WHEN query LIKE 'autovacuum: %' THEN 'autovacuum worker' ELSE 'client backend' END, waiting AS waiting_for_lock,"
const backendCountsSQLpg96OptionalFields = "CASE WHEN query LIKE 'autovacuum: %' THEN 'autovacuum worker' ELSE 'client backend' END, COALESCE(wait_event_type, '') = 'Lock' AS waiting_for_lock,"
const backendCountsSQLpg10OptionalFields = "COALESCE(backend_type, 'unknown'), COALESCE(wait_event_type, '') = 'Lock' AS waiting_for_lock,"

const backendCountsSQL string = `
 SELECT datid,
				usesysid,
				COALESCE(state, 'unknown'),
				%s
				COUNT(*)
	 FROM %s
	GROUP BY 1, 2, 3, 4, 5`

func GetBackendCounts(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresBackendCount, error) {
	var optionalFields string
	var sourceTable string

	if postgresVersion.Numeric >= state.PostgresVersion10 {
		optionalFields = backendCountsSQLpg10OptionalFields
	} else if postgresVersion.Numeric >= state.PostgresVersion96 {
		optionalFields = backendCountsSQLpg96OptionalFields
	} else {
		optionalFields = backendCountsSQLDefaultOptionalFields
	}

	if statsHelperExists(db, "get_stat_activity") {
		sourceTable = "pganalyze.get_stat_activity()"
	} else {
		sourceTable = "pg_stat_activity"
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(backendCountsSQL, optionalFields, sourceTable))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var backendCounts []state.PostgresBackendCount

	for rows.Next() {
		var row state.PostgresBackendCount

		err := rows.Scan(&row.DatabaseOid, &row.RoleOid, &row.State, &row.BackendType,
			&row.WaitingForLock, &row.Count)
		if err != nil {
			return nil, err
		}

		backendCounts = append(backendCounts, row)
	}

	return backendCounts, nil
}
