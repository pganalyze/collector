package dbstats

import (
	"database/sql"

	"github.com/lfittl/pg_query_go"
	"github.com/pganalyze/collector/util"

	null "gopkg.in/guregu/null.v2"
)

type Activity struct {
	Pid             int            `json:"pid"`
	Username        null.String    `json:"username"`
	ApplicationName null.String    `json:"application_name"`
	ClientAddr      null.String    `json:"client_addr"`
	BackendStart    util.Timestamp `json:"backend_start"`
	XactStart       util.Timestamp `json:"xact_start"`
	QueryStart      util.Timestamp `json:"query_start"`
	StateChange     util.Timestamp `json:"state_change"`
	Waiting         null.Bool      `json:"waiting"`
	State           null.String    `json:"state"`
	NormalizedQuery null.String    `json:"normalized_query"`
}

// http://www.postgresql.org/docs/devel/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
const activitySQL string = `SELECT pid, usename, application_name, client_addr::text, backend_start,
				xact_start, query_start, state_change, waiting, state, query
	 FROM pg_stat_activity
	WHERE pid <> pg_backend_pid() AND datname = current_database()`

func GetActivity(logger *util.Logger, db *sql.DB, postgresVersion PostgresVersion) ([]Activity, error) {
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

	var activities []Activity

	for rows.Next() {
		var row Activity
		var query null.String

		err := rows.Scan(&row.Pid, &row.Username, &row.ApplicationName, &row.ClientAddr,
			&row.BackendStart, &row.XactStart, &row.QueryStart, &row.StateChange,
			&row.Waiting, &row.State, &query)
		if err != nil {
			return nil, err
		}

		if !query.IsZero() {
			normalizedQuery, err := pg_query.Normalize(*query.Ptr())
			if err != nil {
				logger.PrintVerbose("Failed to normalize query: %s", err)
			} else {
				row.NormalizedQuery = null.StringFrom(normalizedQuery)
			}
		}

		activities = append(activities, row)
	}

	return activities, nil
}
