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
}

type snapshotPostgres struct {
	Relations []dbstats.Relation `json:"schema"`
}

func collectStatistics(config config.Config, db *sql.DB, dryRun bool) (err error) {
	var stats snapshot

	stats.ActiveQueries = dbstats.GetActivity(db)
	stats.Statements = dbstats.GetStatements(db)
	stats.Postgres.Relations = dbstats.GetRelations(db)
	stats.System = systemstats.GetSystemSnapshot(config)

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

func connectToDb(config config.Config) *sql.DB {
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

func main() {
	var dryRun bool

	flag.BoolVar(&dryRun, "dry-run", false, "Print JSON data that would get sent to web service and exit afterwards.")
	flag.Parse()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}

	config, err := config.Read()
	panicOnErr(err)

	schedulerGroups, err := scheduler.ReadSchedulerGroups(scheduler.DefaultConfig)
	panicOnErr(err)

	db := connectToDb(config)
	defer db.Close()

	// Initial run to ensure everything is working
	err = collectStatistics(config, db, dryRun)
	panicOnErr(err)

	if dryRun {
		return
	}

	stop := schedulerGroups["stats"].Schedule(func() {
		wg.Add(1)

		err := collectStatistics(config, db, false)
		if err != nil {
			// TODO(LukasFittl): We could consider re-running on error (e.g. if it was a temporary server issue)
			log.Print(err)
		}

		wg.Done()
	})

	<-sigs

	signal.Stop(sigs)

	log.Printf("Exiting...")
	stop <- true

	wg.Wait()
}
