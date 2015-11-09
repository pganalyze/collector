package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
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

	"github.com/go-ini/ini"

	"github.com/lfittl/pganalyze-collector-next/dbstats"
	scheduler "github.com/lfittl/pganalyze-collector-next/scheduler"
	systemstats "github.com/lfittl/pganalyze-collector-next/systemstats"
)

func panicOnErr(err error) {
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

	AwsAccessKeyId     string `ini:"aws_access_key_id"`
	AwsSecretAccessKey string `ini:"aws_secret_access_key"`
}

type snapshot struct {
	ActiveQueries []dbstats.Activity  `json:"backends"`
	Statements    []dbstats.Statement `json:"queries"`
	Postgres      snapshotPostgres    `json:"postgres"`
	System        systemstats.SnapshotSystem `json:"system"`
}

type snapshotPostgres struct {
	Relations []dbstats.Relation `json:"schema"`
}

func collectStatistics(config connectionConfig, db *sql.DB) (err error) {
	var stats snapshot

	stats.ActiveQueries = dbstats.GetActivity(db)
	stats.Statements = dbstats.GetStatements(db)
	stats.Postgres.Relations = dbstats.GetRelations(db)
	stats.System = systemstats.GetFromAws(config.AwsAccessKeyId, config.AwsSecretAccessKey)

	statsJSON, _ := json.Marshal(stats)

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

func readConfig() connectionConfig {
	config := &connectionConfig{
		APIURL: "https://api.pganalyze.com/v1/snapshots",
		DbHost: "localhost",
		DbPort: 5432,
	}

	usr, err := user.Current()
	panicOnErr(err)

	filename := usr.HomeDir + "/.pganalyze_collector.conf"

	if _, err := os.Stat(filename); err == nil {
		configFile, err := ini.Load(filename)
		panicOnErr(err)

		err = configFile.Section("pganalyze").MapTo(config)
		panicOnErr(err)
	}

	// The environment variables always trump everything else, and are the default way
	// to configure when running inside a Docker container.
	if apiKey := os.Getenv("PGA_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if apiURL := os.Getenv("PGA_API_URL"); apiURL != "" {
		config.APIURL = apiURL
	}
	if dbURL := os.Getenv("DB_URL"); dbURL != "" {
		config.DbURL = dbURL
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		config.DbName = dbName
	}
	if dbUsername := os.Getenv("DB_USERNAME"); dbUsername != "" {
		config.DbUsername = dbUsername
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		config.DbPassword = dbPassword
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.DbHost = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		config.DbPort, _ = strconv.Atoi(dbPort)
	}

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
	panicOnErr(err)

	err = db.Ping()
	panicOnErr(err)

	return db
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}

	config := readConfig()
	schedulerGroups, err := scheduler.ReadSchedulerGroups(scheduler.DefaultConfig)
	panicOnErr(err)

	db := connectToDb(config)
	defer db.Close()

	// Initial run to ensure everything is working
	err = collectStatistics(config, db)
	panicOnErr(err)

	stop := schedulerGroups["stats"].Schedule(func() {
		wg.Add(1)

		err := collectStatistics(config, db)
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
