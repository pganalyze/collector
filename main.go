package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"sync"
	"syscall"

	"database/sql"

	_ "github.com/lib/pq" // Enable database package to use Postgres

	"github.com/go-ini/ini"

	"github.com/lfittl/pganalyze-collector-next/dbstats"
	scheduler "github.com/lfittl/pganalyze-collector-next/scheduler"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

type connectionConfig struct {
	APIKey     string `ini:"api_key"`
	APIURL     string `ini:"api_url"`
	DbURL      string `ini:"db_url"`
	DbName     string `ini:"db_name"`
	DbUsername string `ini:"db_username"`
	DbPassword string `ini:"db_password"`
	DbHost     string `ini:"db_host"`
	DbPort     int    `ini:"db_port"`
}

type snapshot struct {
	ActiveQueries []dbstats.Activity  `json:"active_queries"`
	Relations     []dbstats.Relation  `json:"relations"`
	Statements    []dbstats.Statement `json:"statements"`
}

func collectStatistics(config connectionConfig, db *sql.DB) {
	var stats snapshot

	stats.ActiveQueries = dbstats.GetActivity(db)
	stats.Statements = dbstats.GetStatements(db)
	stats.Relations = dbstats.GetRelations(db)

	statsJSON, _ := json.Marshal(stats)
	fmt.Println(string(statsJSON))
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

	if config.DbURL != "" {
		dbinfo = config.DbURL
	} else {
		dbinfo = fmt.Sprintf("user=%s dbname=%s host=%s port=%d sslmode=disable connect_timeout=10",
			config.DbUsername, config.DbName, config.DbHost, config.DbPort)

		if config.DbPassword != "" {
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
	schedulerGroups, err := scheduler.ReadSchedulerGroups(scheduler.DefaultConfig)
	if err != nil {
		panic("Could not read scheduler configuration - please make sure scheduler.toml exists")
	}

	db := connectToDb(config)
	defer db.Close()

	// Initial run to ensure everything is working
	collectStatistics(config, db)

	stop := schedulerGroups["stats"].Schedule(func() {
		wg.Add(1)
		collectStatistics(config, db)
		wg.Done()
	})

	<-sigs

	signal.Stop(sigs)

	log.Printf("Exiting...")
	stop <- true

	wg.Wait()
}
