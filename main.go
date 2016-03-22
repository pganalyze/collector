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
	"os/user"
	"strconv"
	"sync"
	"syscall"
	"time"

	"database/sql"

	_ "github.com/lib/pq" // Enable database package to use Postgres

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/dbstats"
	"github.com/pganalyze/collector/explain"
	"github.com/pganalyze/collector/logs"
	scheduler "github.com/pganalyze/collector/scheduler"
	systemstats "github.com/pganalyze/collector/systemstats"
)

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

	stats.ActiveQueries, err = dbstats.GetActivity(db)
	if err != nil {
		return err
	}

	stats.Statements, err = dbstats.GetStatements(db)
	if err != nil {
		return err
	}

	stats.Postgres.Relations, err = dbstats.GetRelations(db)
	if err != nil {
		return err
	}

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
		"submitter":          {"pganalyze-collector 0.9.0rc1"},
		"system_information": {"false"},
		"no_reset":           {"true"},
		"query_source":       {"pg_stat_statements"},
		"collected_at":       {fmt.Sprintf("%d", time.Now().Unix())},
	})
	// TODO: We could consider re-running on error (e.g. if it was a temporary server issue)
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

	log.Printf("[%s] Submitted snapshot successfully", config.SectionName)
	return
}

func connectToDb(config config.DatabaseConfig) (*sql.DB, error) {
	var dbinfo string

	if config.DbURL != "" {
		dbinfo = config.DbURL
	} else {
		dbinfo = fmt.Sprintf("user=%s dbname=%s host=%s port=%d connect_timeout=10",
			config.DbUsername, config.DbName, config.DbHost, config.DbPort)

		if config.DbPassword != "" {
			dbinfo += fmt.Sprintf(" password=%s", config.DbPassword)
		}
	}

	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

type ConfigAndConnection struct {
	config     config.DatabaseConfig
	connection *sql.DB
}

func run(wg sync.WaitGroup, dryRun bool, configFilename string) chan<- bool {
	var databases []ConfigAndConnection

	schedulerGroups, err := scheduler.ReadSchedulerGroups(scheduler.DefaultConfig)
	if err != nil {
		log.Print("Error: Could not read scheduler groups, awaiting SIGHUP or process kill")
		return nil
	}

	databaseConfigs, err := config.Read(configFilename)
	if err != nil {
		log.Print("Error: Could not read configuration, awaiting SIGHUP or process kill")
		return nil
	}

	for _, config := range databaseConfigs {
		database := ConfigAndConnection{config: config}
		database.connection, err = connectToDb(config)
		if err != nil {
			log.Printf("[%s] Error: Failed to connect to database: %s", config.SectionName, err)
		} else {
			databases = append(databases, database)
		}
	}

	// Initial run to ensure everything is working
	for _, database := range databases {
		err = collectStatistics(database.config, database.connection, dryRun)
		if err != nil {
			log.Printf("[%s] %s", database.config.SectionName, err)
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
				log.Printf("[%s] %s", database.config.SectionName, err)
			}
		}

		wg.Done()
	})

	return stop
}

func main() {
	var dryRun bool
	var configFilename string
	var pidFilename string

	usr, err := user.Current()
	if err != nil {
		log.Print("Could not get user context from operating system - can't initialize, exiting.")
		return
	}

	flag.BoolVar(&dryRun, "dry-run", false, "Print JSON data that would get sent to web service and exit afterwards.")
	flag.StringVar(&configFilename, "config", usr.HomeDir+"/.pganalyze_collector.conf", "Specifiy alternative path for config file.")
	flag.StringVar(&pidFilename, "pidfile", "", "Specifies a path that a pidfile should be written to. (default is no pidfile being written)")
	flag.Parse()

	if pidFilename != "" {
		pid := os.Getpid()
		err := ioutil.WriteFile(pidFilename, []byte(strconv.Itoa(pid)), 0644)
		if err != nil {
			log.Printf("Could not write pidfile to \"%s\" as requested", pidFilename)
			return
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	wg := sync.WaitGroup{}

ReadConfigAndRun:
	stop := run(wg, dryRun, configFilename)
	if dryRun {
		return
	}

	// Block here until we get any of the registered signals
	s := <-sigs

	// Stop the scheduled runs
	stop <- true

	if s == syscall.SIGHUP {
		log.Print("Reloading configuration...")
		goto ReadConfigAndRun
	}

	signal.Stop(sigs)

	log.Print("Exiting...")
	wg.Wait()
}
