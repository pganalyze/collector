package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"database/sql"

	_ "github.com/lib/pq" // Enable database package to use Postgres

	"github.com/lfittl/pganalyze-collector-next/config"
	"github.com/lfittl/pganalyze-collector-next/dbstats"
	"github.com/lfittl/pganalyze-collector-next/explain"
	"github.com/lfittl/pganalyze-collector-next/logs"
	scheduler "github.com/lfittl/pganalyze-collector-next/scheduler"
	systemstats "github.com/lfittl/pganalyze-collector-next/systemstats"
)

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

type snapshot struct {
	ActiveQueries []dbstats.Activity          `json:"backends"`
	Statements    []dbstats.Statement         `json:"queries"`
	Postgres      snapshotPostgres            `json:"postgres"`
	System        *systemstats.SystemSnapshot `json:"system"`
	Logs          []logs.Line                 `json:"logs"`
	Explains      []explain.Explain           `json:"explains"`
}

type snapshotPostgres struct {
	Relations []dbstats.Relation `json:"schema"`
}

func collectStatistics(config config.DatabaseConfig, db *sql.DB, dryRun bool) (err error) {
	var stats snapshot
	var explainInputs []explain.ExplainInput

	stats.ActiveQueries = dbstats.GetActivity(db)

	stats.Statements, err = dbstats.GetStatements(db)
	if err != nil {
		return err
	}

	stats.Postgres.Relations = dbstats.GetRelations(db)
	stats.System = systemstats.GetSystemSnapshot(config)
	stats.Logs, explainInputs = logs.GetLogLines(config)

	stats.Explains = explain.RunExplain(db, explainInputs)

	statsJSON, _ := json.Marshal(stats)

	if dryRun {
		var out bytes.Buffer
		json.Indent(&out, statsJSON, "", "\t")
		log.Printf("Dry run - JSON data that would have been sent:\n%s", out.String())
		return
	}

	var compressedJSON bytes.Buffer
	w := zlib.NewWriter(&compressedJSON)
	w.Write(statsJSON)
	w.Close()

	resp, err := http.PostForm(config.APIURL, url.Values{
		"data":               {compressedJSON.String()},
		"data_compressor":    {"zlib"},
		"api_key":            {config.APIKey},
		"submitter":          {"pganalyze-collector-next"},
		"system_information": {"false"},
		"no_reset":           {"true"},
		"query_source":       {"pg_stat_statements"},
		"collected_at":       {fmt.Sprintf("%d", time.Now().Unix())},
	})
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Error when submitting: %s\n", body)
		return
	}

	log.Printf("Submitted snapshot successfully\n")
	return
}

func connectToDb(config config.DatabaseConfig) *sql.DB {
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
	panicOnErr(err)

	err = db.Ping()
	panicOnErr(err)

	return db
}

type ConfigAndConnection struct {
	config     config.DatabaseConfig
	connection *sql.DB
}

func run(wg sync.WaitGroup, dryRun bool) chan<- bool {
	var databases []ConfigAndConnection

	schedulerGroups, err := scheduler.ReadSchedulerGroups(scheduler.DefaultConfig)
	panicOnErr(err)

	databaseConfigs, err := config.Read()
	panicOnErr(err)

	for _, config := range databaseConfigs {
		database := ConfigAndConnection{config: config}

		database.connection = connectToDb(databaseConfigs[0])
		defer database.connection.Close()

		databases = append(databases, database)
	}

	// Initial run to ensure everything is working
	for _, database := range databases {
		err = collectStatistics(database.config, database.connection, dryRun)
		if err != nil {
			log.Print(err)
		}
	}

	if dryRun {
		return nil
	}

	stop := schedulerGroups["stats"].Schedule(func() {
		wg.Add(1)

		for _, database := range databases {
			err := collectStatistics(database.config, database.connection, false)
			if err != nil {
				// TODO(LukasFittl): We could consider re-running on error (e.g. if it was a temporary server issue)
				log.Print(err)
			}
		}

		wg.Done()
	})

	return stop
}

func main() {
	var dryRun bool

	flag.BoolVar(&dryRun, "dry-run", false, "Print JSON data that would get sent to web service and exit afterwards.")
	flag.Parse()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	wg := sync.WaitGroup{}

ReadConfigAndRun:
	stop := run(wg, dryRun)
	if stop == nil {
		return
	}

	s := <-sigs

	if s == syscall.SIGHUP {
		log.Printf("Reloading configuration...")
		goto ReadConfigAndRun
	}

	signal.Stop(sigs)

	log.Printf("Exiting...")
	stop <- true

	wg.Wait()
}
