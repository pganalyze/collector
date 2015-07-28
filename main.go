package main

import (
  "encoding/json"
  "fmt"
  "log"
  "time"
  "os"
  "os/signal"
  "sync"
  "syscall"

  "database/sql"
  _ "github.com/lib/pq"

  "github.com/pganalyze/agent/dbstats"
)

func checkErr(err error) {
  if err != nil {
    panic(err)
  }
}

type snapshot struct {
  ActiveQueries []dbstats.StatActivity `json:"active_queries"`
  Relations []dbstats.Relation `json:"relations"`
}

func checkStatActivity() {
  DB_USER := "lfittl"
  DB_NAME := "pganalyze"

  dbinfo := fmt.Sprintf("user=%s dbname=%s sslmode=disable",
                        DB_USER, DB_NAME)
  db, err := sql.Open("postgres", dbinfo)
  checkErr(err)
  defer db.Close()

  var stats snapshot

  stats.ActiveQueries = dbstats.GetStatActivity(db)
  stats.Relations = dbstats.GetRelations(db)

  statsJson, _ := json.Marshal(stats)
  fmt.Println(string(statsJson))
}

func main() {
  sigs := make(chan os.Signal, 1)
  signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

  wg := sync.WaitGroup{}

  ticker := time.NewTicker(time.Millisecond * 1000)

  go func() {
    for _ = range ticker.C {
      wg.Add(1)
      checkStatActivity()
      wg.Done()
    }
  }()

  <-sigs

  signal.Stop(sigs)

  log.Printf("Exiting...")
  ticker.Stop()

  wg.Wait()
}
