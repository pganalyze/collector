package dbstats

import (
	"database/sql"

	null "gopkg.in/guregu/null.v2"
)

type Activity struct {
	Pid             int         `json:"pid"`
	Username        string      `json:"username"`
	ApplicationName string      `json:"application_name"`
	ClientAddr      null.String `json:"client_addr"`
	BackendStart    Timestamp   `json:"backend_start"`
	XactStart       Timestamp   `json:"xact_start"`
	QueryStart      Timestamp   `json:"query_start"`
	StateChange     Timestamp   `json:"state_change"`
	Waiting         null.Bool   `json:"waiting"`
	State           string      `json:"state"`
}

// http://www.postgresql.org/docs/devel/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
//
// Note: We don't include query to avoid sending sensitive data
const activitySQL string = `SELECT pid, usename, application_name, client_addr::text, backend_start,
				xact_start, query_start, state_change, waiting, state
	 FROM pg_stat_activity
	WHERE pid <> pg_backend_pid() AND datname = current_database()`

func GetActivity(db *sql.DB) []Activity {
	stmt, err := db.Prepare(queryMarkerSQL + activitySQL)
	checkErr(err)

	defer stmt.Close()

	rows, err := stmt.Query()
	checkErr(err)
	defer rows.Close()

	var activities []Activity

	for rows.Next() {
		var row Activity

		err := rows.Scan(&row.Pid, &row.Username, &row.ApplicationName, &row.ClientAddr,
			&row.BackendStart, &row.XactStart, &row.QueryStart, &row.StateChange,
			&row.Waiting, &row.State)
		checkErr(err)

		activities = append(activities, row)
	}

	return activities
}
