package dbstats

import (
  "time"
  "database/sql"
)

type StatActivity struct {
  Pid int `json:"pid"`
  Usename string `json:"query"`
  ApplicationName string `json:"state"`
  ClientAddr string `json:"client_addr"`
  BackendStart time.Time `json:"backend_start"`
  XactStart time.Time `json:"xact_start"`
  QueryStart time.Time `json:"query_start"`
  StateChange time.Time `json:"state_change"`
  Waiting bool `json:"waiting"`
  State string `json:"state"`
}

const statActivitySQL string =
`SELECT pid, usename, application_name, client_addr::text, backend_start,
        xact_start, query_start, state_change, waiting, state
   FROM pg_stat_activity
  WHERE datname = current_database()`
// pid <> pg_backend_pid() AND

func GetStatActivity(db *sql.DB) []StatActivity {
  stmt, err := db.Prepare(statActivitySQL)
  checkErr(err)

  defer stmt.Close()

  rows, err := stmt.Query()

  var activities []StatActivity

  defer rows.Close()
  for rows.Next() {
    var row StatActivity

    err := rows.Scan(&row.Pid, &row.Usename, &row.ApplicationName, &row.ClientAddr,
                     &row.BackendStart, &row.XactStart, &row.QueryStart, &row.StateChange,
                     &row.Waiting, &row.State)
    checkErr(err)

    activities = append(activities, row)
  }

  return activities
}
