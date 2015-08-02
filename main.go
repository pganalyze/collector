package main

import (
  "encoding/json"
  "fmt"
  "log"
  "time"
  "os"
  "os/signal"
  "os/user"
  "sync"
  "syscall"

  "database/sql"
  _ "github.com/lib/pq"

  "github.com/go-ini/ini"

  "github.com/pganalyze/agent/dbstats"
)

func checkErr(err error) {
  if err != nil {
    panic(err)
  }
}

type connectionConfig struct {
  ApiKey string `ini:"api_key"`
  ApiUrl string `ini:"api_url"`
  DbUrl string `ini:"db_url"`
  DbName string `ini:"db_name"`
  DbUsername string `ini:"db_username"`
  DbPassword string `ini:"db_password"`
  DbHost string `ini:"db_host"`
  DbPort int `ini:"db_port"`
}

type snapshot struct {
  ActiveQueries []dbstats.Activity `json:"active_queries"`
  Relations []dbstats.Relation `json:"relations"`
  Statements []dbstats.Statement `json:"statements"`
}

func collectStatistics(config connectionConfig, db *sql.DB) {
  var stats snapshot

  //stats.ActiveQueries = dbstats.GetActivity(db)
  stats.Statements = dbstats.GetStatements(db)
  //stats.Relations = dbstats.GetRelations(db)

  statsJson, _ := json.Marshal(stats)
  fmt.Println(string(statsJson))
}

func readConfig() connectionConfig {
  config := &connectionConfig{
    DbHost: "localhost",
    DbPort: 5432,
  }

  usr, err := user.Current()
  checkErr(err)

  configFile, err := ini.Load(usr.HomeDir + "/.pganalyze_collector.conf")
  checkErr(err)

  err = configFile.Section("pganalyze").MapTo(config)
  checkErr(err)

  return *config
}

func connectToDb(config connectionConfig) *sql.DB {
  var dbinfo string

  if (config.DbUrl != "") {
    dbinfo = config.DbUrl
  } else {
    dbinfo = fmt.Sprintf("user=%s dbname=%s host=%s port=%d sslmode=disable connect_timeout=10",
                          config.DbUsername, config.DbName, config.DbHost, config.DbPort)

    if (config.DbPassword != "") {
      dbinfo += fmt.Sprintf(" password=%s", config.DbPassword)
    }
  }

  db, err := sql.Open("postgres", dbinfo)
  checkErr(err)

  err = db.Ping()
  checkErr(err)

  return db
}

func main() {
  sigs := make(chan os.Signal, 1)
  signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

  wg := sync.WaitGroup{}

  config := readConfig()

  db := connectToDb(config)
  defer db.Close()

  // Initial run to ensure everything is working
  collectStatistics(config, db)

  ticker := time.NewTicker(time.Millisecond * 10000)

  go func() {
    for _ = range ticker.C {
      wg.Add(1)
      collectStatistics(config, db)
      wg.Done()
    }
  }()

  <-sigs

  signal.Stop(sigs)

  log.Printf("Exiting...")
  ticker.Stop()

  wg.Wait()
}
