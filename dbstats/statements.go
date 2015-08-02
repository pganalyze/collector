package dbstats

import (
  "database/sql"
  "fmt"
  null "gopkg.in/guregu/null.v2"
)

type Statement struct {
  Username string `json:"username"`
  Query string `json:"query"`
  Calls int64 `json:"calls"`
  TotalTime float64 `json:"total_time"`
  Rows int64 `json:"rows"`
  SharedBlksHit int64 `json:"shared_blks_hit"`
  SharedBlksRead int64 `json:"shared_blks_read"`
  SharedBlksDirtied int64 `json:"shared_blks_dirtied"`
  SharedBlksWritten int64 `json:"shared_blks_written"`
  LocalBlksHit int64 `json:"local_blks_hit"`
  LocalBlksRead int64 `json:"local_blks_read"`
  LocalBlksDirtied int64 `json:"local_blks_dirtied"`
  LocalBlksWritten int64 `json:"local_blks_written"`
  TempBlksRead int64 `json:"temp_blks_read"`
  TempBlksWritten int64 `json:"temp_blks_written"`
  BlkReadTime float64 `json:"blk_read_time"`
  BlkWriteTime float64 `json:"blk_write_time"`

  // Postgres 9.4+
  Queryid null.Int `json:"query_id"`

  // Postgres 9.5+
  MinTime null.Float `json:"min_time"`
  MaxTime null.Float `json:"max_time"`
  MeanTime null.Float `json:"mean_time"`
  StddevTime null.Float `json:"stddev_time"`
}

const pg93OptionalFields = "NULL, NULL, NULL, NULL, NULL"
const pg94OptionalFields = "queryid, NULL, NULL, NULL, NULL"
const pg95OptionalFields = "queryid, min_time, max_time, mean_time, stddev_time"

const statementSQL string =
`SELECT (SELECT rolname FROM pg_authid WHERE oid = userid) AS username,
        query, calls, total_time, rows, shared_blks_hit, shared_blks_read,
        shared_blks_dirtied, shared_blks_written, local_blks_hit,
        local_blks_read, local_blks_dirtied, local_blks_written,
        temp_blks_read, temp_blks_written, blk_read_time, blk_write_time,
        %s
   FROM pg_stat_statements
  WHERE dbid IN (SELECT oid FROM pg_database WHERE datname = current_database())`

func GetStatements(db *sql.DB) []Statement {
  // TODO(LukasFittl): Use correct optional fields based on version
  stmt, err := db.Prepare(fmt.Sprintf(statementSQL, pg94OptionalFields))
  checkErr(err)

  defer stmt.Close()

  rows, err := stmt.Query()

  var statements []Statement

  defer rows.Close()
  for rows.Next() {
    var row Statement

    err := rows.Scan(&row.Username, &row.Query, &row.Calls, &row.TotalTime, &row.Rows,
                     &row.SharedBlksHit, &row.SharedBlksRead, &row.SharedBlksDirtied, &row.SharedBlksWritten,
                     &row.LocalBlksHit, &row.LocalBlksRead, &row.LocalBlksDirtied, &row.LocalBlksWritten,
                     &row.TempBlksRead, &row.TempBlksWritten, &row.BlkReadTime, &row.BlkWriteTime,
                     &row.Queryid, &row.MinTime, &row.MaxTime, &row.MeanTime, &row.StddevTime)
    checkErr(err)

    statements = append(statements, row)
  }

  return statements
}
