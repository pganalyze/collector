package postgres

import (
	"database/sql"

	"github.com/lfittl/pg_query_go"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	null "gopkg.in/guregu/null.v3"
)

// http://www.postgresql.org/docs/devel/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
const activitySQL string = `SELECT pid, usename, application_name, client_addr::text, backend_start,
				xact_start, query_start, state_change, waiting, state, query
	 FROM pg_stat_activity
	WHERE pid <> pg_backend_pid() AND datname = current_database()`

func GetBackends(logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion) ([]state.PostgresBackend, error) {
	stmt, err := db.Prepare(QueryMarkerSQL + activitySQL)
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

		err := rows.Scan(&row.Pid, &row.Username, &row.ApplicationName, &row.ClientAddr,
			&row.BackendStart, &row.XactStart, &row.QueryStart, &row.StateChange,
			&row.Waiting, &row.State, &query)
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
