package postgres

import (
	"database/sql"
	"fmt"

	"github.com/guregu/null"
	"github.com/lfittl/pg_query_go"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const activitySQLDefaultOptionalFields = "waiting, NULL, NULL"
const activitySQLpg94OptionalFields = "waiting, backend_xid, backend_xmin"
const activitySQLpg96OptionalFields = "wait_event IS NOT NULL, backend_xid, backend_xmin"

// https://www.postgresql.org/docs/9.5/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
const activitySQL string = `SELECT datid, usesysid, pid, application_name, client_addr::text, client_port,
				backend_start, xact_start, query_start, state_change, %s, state, query
	 FROM pg_stat_activity
	WHERE pid <> pg_backend_pid() AND datname = current_database()`

func GetBackends(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresBackend, error) {
	var optionalFields string

	if postgresVersion.Numeric >= state.PostgresVersion96 {
		optionalFields = activitySQLpg96OptionalFields
	} else if postgresVersion.Numeric >= state.PostgresVersion94 {
		optionalFields = activitySQLpg94OptionalFields
	} else {
		optionalFields = activitySQLDefaultOptionalFields
	}

	stmt, err := db.Prepare(QueryMarkerSQL + fmt.Sprintf(activitySQL, optionalFields))
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var activities []state.PostgresBackend

	for rows.Next() {
		var row state.PostgresBackend
		var query null.String

		err := rows.Scan(&row.DatabaseOid, &row.UserOid, &row.Pid, &row.ApplicationName,
			&row.ClientAddr, &row.ClientPort, &row.BackendStart, &row.XactStart, &row.QueryStart,
			&row.StateChange, &row.Waiting, &row.BackendXid, &row.BackendXmin, &row.State, &query)
		if err != nil {
			return nil, err
		}

		if query.Valid && query.String != "<insufficient privilege>" {
			normalizedQuery, err := pg_query.Normalize(query.String)
			if err != nil {
				logger.PrintVerbose("Failed to normalize query, excluding from statistics: %s", err)
			} else {
				row.NormalizedQuery = null.StringFrom(normalizedQuery)
			}
		}

		activities = append(activities, row)
	}

	return activities, nil
}
