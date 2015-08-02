package dbstats

import (
  "time"
  "database/sql"
)

type Activity struct {
  Pid int `json:"pid"`
  Username string `json:"username"`
  ApplicationName string `json:"application_name"`
  ClientAddr string `json:"client_addr"`
  BackendStart time.Time `json:"backend_start"`
  XactStart time.Time `json:"xact_start"`
  QueryStart time.Time `json:"query_start"`
  StateChange time.Time `json:"state_change"`
  Waiting bool `json:"waiting"`
  State string `json:"state"`
}

const activitySQL string =
`SELECT pid, usename, application_name, client_addr::text, backend_start,
        xact_start, query_start, state_change, waiting, state
   FROM pg_stat_activity
  WHERE pid <> pg_backend_pid() AND datname = current_database()`

func GetActivity(db *sql.DB) []Activity {
  stmt, err := db.Prepare(activitySQL)
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
