package dbstats

import (
	"database/sql"

	"github.com/lfittl/pg_query_go"
	"github.com/pganalyze/collector/snapshot"
	"github.com/pganalyze/collector/util"

	null "gopkg.in/guregu/null.v2"
)

// http://www.postgresql.org/docs/devel/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
const activitySQL string = `SELECT pid, usename, application_name, client_addr::text, backend_start,
				xact_start, query_start, state_change, waiting, state, query
	 FROM pg_stat_activity
	WHERE pid <> pg_backend_pid() AND datname = current_database()`

func GetActivity(logger *util.Logger, db *sql.DB, postgresVersion snapshot.PostgresVersion) ([]*snapshot.Activity, error) {
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

	var activities []*snapshot.Activity

	for rows.Next() {
		var row snapshot.Activity
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
				row.NormalizedQuery = normalizedQuery
			}
		}

		activities = append(activities, &row)
	}

	return activities, nil
}
