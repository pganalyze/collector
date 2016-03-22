package dbstats

import (
	"database/sql"

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
}

// http://www.postgresql.org/docs/devel/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
//
// Note: We don't include query to avoid sending sensitive data
const activitySQL string = `SELECT pid, usename, application_name, client_addr::text, backend_start,
				xact_start, query_start, state_change, waiting, state
	 FROM pg_stat_activity
	WHERE pid <> pg_backend_pid() AND datname = current_database()`

func GetActivity(db *sql.DB) ([]Activity, error) {
	stmt, err := db.Prepare(queryMarkerSQL + activitySQL)
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

		err := rows.Scan(&row.Pid, &row.Username, &row.ApplicationName, &row.ClientAddr,
			&row.BackendStart, &row.XactStart, &row.QueryStart, &row.StateChange,
			&row.Waiting, &row.State)
		if err != nil {
			return nil, err
		}

		activities = append(activities, row)
	}

	return activities, nil
}
