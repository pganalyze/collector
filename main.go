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
	"strings"
	"sync"
	"syscall"
	"time"

	flag "github.com/ogier/pflag"

	"database/sql"

	_ "github.com/lib/pq" // Enable database package to use Postgres

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/dbstats"
	"github.com/pganalyze/collector/explain"
	"github.com/pganalyze/collector/logs"
	scheduler "github.com/pganalyze/collector/scheduler"
	systemstats "github.com/pganalyze/collector/systemstats"
	"github.com/pganalyze/collector/util"
)

type snapshot struct {
	ActiveQueries []dbstats.Activity          `json:"backends"`
	Statements    []dbstats.Statement         `json:"queries"`
	Postgres      snapshotPostgres            `json:"postgres"`
	System        *systemstats.SystemSnapshot `json:"system"`
	Logs          []logs.Line                 `json:"logs"`
	Explains      []explain.Explain           `json:"explains"`
	Opts          snapshotOpts                `json:"opts"`
}

type snapshotOpts struct {
	StatementStatsAreDiffed        bool `json:"statement_stats_are_diffed"`
	PostgresRelationStatsAreDiffed bool `json:"postgres_relation_stats_are_diffed"`
}

type snapshotPostgres struct {
	Relations []dbstats.Relation      `json:"schema"`
	Settings  []dbstats.Setting       `json:"settings"`
	Functions []dbstats.Function      `json:"functions"`
	Version   dbstats.PostgresVersion `json:"version"`
}

type collectionOpts struct {
	collectPostgresRelations bool
	collectPostgresSettings  bool
	collectPostgresLocks     bool
	collectPostgresFunctions bool
	collectPostgresBloat     bool
	collectPostgresViews     bool

	collectLogs              bool
	collectExplain           bool
	collectSystemInformation bool

	collectorApplicationName string

	diffStatements bool

	submitCollectedData bool
	testRun             bool
}

type statementStatsKey struct {
	Userid  int
	Queryid int
}

type statsState struct {
	statements map[statementStatsKey]dbstats.Statement
}

type database struct {
	config           config.DatabaseConfig
	connection       *sql.DB
	prevState        statsState
	requestedSslMode string
}

func collectStatistics(db database, collectionOpts collectionOpts, logger *util.Logger) (newState statsState, err error) {
	var stats snapshot
	var explainInputs []explain.ExplainInput

	postgresVersion, err := dbstats.GetPostgresVersion(logger, db.connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	stats.Postgres.Version = postgresVersion

	if postgresVersion.Numeric < dbstats.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required.", postgresVersion.Short)
		return
	}

	stats.ActiveQueries, err = dbstats.GetActivity(logger, db.connection, postgresVersion)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_activity")
		return
	}

	stats.Statements, err = dbstats.GetStatements(logger, db.connection, postgresVersion)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_statements")
		return
	}

	if collectionOpts.diffStatements && postgresVersion.Numeric >= dbstats.PostgresVersion94 {
		var diffedStatements []dbstats.Statement

		newState.statements = make(map[statementStatsKey]dbstats.Statement)

		// Iterate through all statements, and diff them based on the previous state
		for _, statement := range stats.Statements {
			key := statementStatsKey{Userid: statement.Userid, Queryid: int(statement.Queryid.Int64)}

			prevStatement, exists := db.prevState.statements[key]
			if exists {
				diffedStatement := statement.DiffSince(prevStatement)
				if diffedStatement.Calls > 0 {
					diffedStatements = append(diffedStatements, diffedStatement)
				}
			} else if len(db.prevState.statements) > 0 {
				diffedStatements = append(diffedStatements, statement)
			}

			newState.statements[key] = statement
		}

		stats.Statements = diffedStatements
		stats.Opts.StatementStatsAreDiffed = true
	}

	if collectionOpts.collectPostgresRelations {
		stats.Postgres.Relations, err = dbstats.GetRelations(db.connection, postgresVersion, collectionOpts.collectPostgresBloat)
		if err != nil {
			logger.PrintError("Error collecting schema information")
			return
		}
	}

	if collectionOpts.collectPostgresSettings {
		stats.Postgres.Settings, err = dbstats.GetSettings(db.connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	if collectionOpts.collectPostgresFunctions {
		stats.Postgres.Functions, err = dbstats.GetFunctions(db.connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting stored procedures")
			return
		}
	}

	if collectionOpts.collectSystemInformation {
		stats.System = systemstats.GetSystemSnapshot(db.config)
	}

	if collectionOpts.collectLogs {
		stats.Logs, explainInputs = logs.GetLogLines(db.config)

		if collectionOpts.collectExplain {
			stats.Explains = explain.RunExplain(db.connection, explainInputs)
		}
	}

	statsJSON, _ := json.Marshal(stats)

	if !collectionOpts.submitCollectedData {
		var out bytes.Buffer
		json.Indent(&out, statsJSON, "", "\t")
		logger.PrintInfo("Dry run - JSON data that would have been sent will be output on stdout:\n")
		fmt.Print(out.String())
		return
	}

	var compressedJSON bytes.Buffer
	w := zlib.NewWriter(&compressedJSON)
	w.Write(statsJSON)
	w.Close()

	requestURL := db.config.APIBaseURL + "/v1/snapshots"

	if collectionOpts.testRun {
		requestURL = db.config.APIBaseURL + "/v1/snapshots/test"
	}

	data := url.Values{
		"data":            {compressedJSON.String()},
		"data_compressor": {"zlib"},
		"api_key":         {db.config.APIKey},
		"submitter":       {"pganalyze-collector 0.9.0rc6"},
		"no_reset":        {"true"},
		"query_source":    {"pg_stat_statements"},
		"collected_at":    {fmt.Sprintf("%d", time.Now().Unix())},
	}

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(data.Encode()))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	resp, err := http.DefaultClient.Do(req)
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

	if len(body) > 0 {
		logger.PrintInfo("%s", body)
	} else {
		logger.PrintInfo("Submitted snapshot successfully")
	}

	return
}

func collectAllDatabases(databases []database, globalCollectionOpts collectionOpts, logger *util.Logger) {
	for idx, db := range databases {
		var err error

		prefixedLogger := logger.WithPrefix(db.config.SectionName)

		db.connection, err = establishConnection(db, logger, globalCollectionOpts)
		if err != nil {
			prefixedLogger.PrintError("Error: Failed to connect to database: %s", err)
			return
		}

		newState, err := collectStatistics(db, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Error: Could not collect data: %s", err)
		} else {
			databases[idx].prevState = newState
		}

		// This is the easiest way to avoid opening multiple connections to different databases on the same instance
		db.connection.Close()
		db.connection = nil
	}
}

func validateConnectionCount(connection *sql.DB, logger *util.Logger, globalCollectionOpts collectionOpts) {
	var connectionCount int

	connection.QueryRow(dbstats.QueryMarkerSQL + "SELECT COUNT(*) FROM pg_stat_activity WHERE application_name = '" + globalCollectionOpts.collectorApplicationName + "'").Scan(&connectionCount)

	if connectionCount > 1 {
		logger.PrintError("Too many open monitoring connections (%d), exiting", connectionCount)
		panic("Too many open monitoring connections")
	}

	return
}

func connectToDb(config config.DatabaseConfig, logger *util.Logger, globalCollectionOpts collectionOpts) (*sql.DB, error) {
	connectString := config.GetPqOpenString()

	if strings.HasPrefix(connectString, "postgres://") || strings.HasPrefix(connectString, "postgresql://") {
		if strings.Contains(connectString, "?") {
			connectString += "&"
		} else {
			connectString += "?"
		}
	} else {
		connectString += " "
	}
	connectString += "application_name=" + globalCollectionOpts.collectorApplicationName

	// logger.PrintVerbose("sql.Open(\"postgres\", \"%s\")", connectString)

	db, err := sql.Open("postgres", connectString)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(30 * time.Second)

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func establishConnection(db database, logger *util.Logger, globalCollectionOpts collectionOpts) (connection *sql.DB, err error) {
	connection, err = connectToDb(db.config, logger, globalCollectionOpts)
	if err != nil {
		if err.Error() == "pq: SSL is not enabled on the server" && db.requestedSslMode == "prefer" {
			db.config.DbSslMode = "disable"
			connection, err = connectToDb(db.config, logger, globalCollectionOpts)
		}
	}

	if err != nil {
		return
	}

	validateConnectionCount(connection, logger, globalCollectionOpts)

	return
}

func run(wg sync.WaitGroup, globalCollectionOpts collectionOpts, logger *util.Logger, configFilename string) chan<- bool {
	var databases []database

	schedulerGroups, err := scheduler.ReadSchedulerGroups(scheduler.DefaultConfig)
	if err != nil {
		logger.PrintError("Error: Could not read scheduler groups, awaiting SIGHUP or process kill")
		return nil
	}

	databaseConfigs, err := config.Read(configFilename)
	if err != nil {
		logger.PrintError("Error: Could not read configuration, awaiting SIGHUP or process kill")
		return nil
	}

	for _, config := range databaseConfigs {
		db := database{config: config, requestedSslMode: config.DbSslMode}

		// Go's lib/pq does not support sslmode properly, so we have to implement the "prefer" mode ourselves
		if db.requestedSslMode == "prefer" {
			db.config.DbSslMode = "require"
		}

		databases = append(databases, db)
	}

	// We intentionally don't do a test-run in the normal mode, since we're fine with
	// a later SIGHUP that fixes the config (or a temporarily unreachable server at start)
	if globalCollectionOpts.testRun {
		collectAllDatabases(databases, globalCollectionOpts, logger)
		return nil
	}

	stop := schedulerGroups["stats"].Schedule(func() {
		wg.Add(1)
		collectAllDatabases(databases, globalCollectionOpts, logger)
		wg.Done()
	}, logger, "collection of all databases")

	return stop
}

func main() {
	var dryRun bool
	var testRun bool
	var configFilename string
	var pidFilename string
	var noPostgresSettings, noPostgresLocks, noPostgresFunctions, noPostgresBloat, noPostgresViews bool
	var noPostgresRelations, noLogs, noExplain, noSystemInformation, diffStatements bool

	logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}

	usr, err := user.Current()
	if err != nil {
		logger.PrintError("Could not get user context from operating system - can't initialize, exiting.")
		return
	}

	flag.BoolVarP(&testRun, "test", "t", false, "Tests whether we can successfully collect data, submits it to the server, and exits afterwards.")
	flag.BoolVarP(&logger.Verbose, "verbose", "v", false, "Outputs additional debugging information, use this if you're encoutering errors or other problems.")
	flag.BoolVar(&dryRun, "dry-run", false, "Print JSON data that would get sent to web service (without actually sending) and exit afterwards.")
	flag.BoolVar(&noPostgresRelations, "no-postgres-relations", false, "Don't collect any Postgres relation information (not recommended)")
	flag.BoolVar(&noPostgresSettings, "no-postgres-settings", false, "Don't collect Postgres configuration settings")
	flag.BoolVar(&noPostgresLocks, "no-postgres-locks", false, "Don't collect Postgres lock information (NOTE: This is always enabled right now, i.e. no lock data is gathered)")
	flag.BoolVar(&noPostgresFunctions, "no-postgres-functions", false, "Don't collect Postgres function/procedure information")
	flag.BoolVar(&noPostgresBloat, "no-postgres-bloat", false, "Don't collect Postgres table/index bloat statistics")
	flag.BoolVar(&noPostgresViews, "no-postgres-views", false, "Don't collect Postgres view/materialized view information (NOTE: This is not implemented right now - views are always collected)")
	flag.BoolVar(&noLogs, "no-logs", false, "Don't collect log data")
	flag.BoolVar(&noExplain, "no-explain", false, "Don't automatically EXPLAIN slow queries logged in the logfile")
	flag.BoolVar(&noSystemInformation, "no-system-information", false, "Don't collect OS level performance data")
	flag.BoolVar(&diffStatements, "diff-statements", false, "Send a diff of the pg_stat_statements statistics, instead of counter values")
	flag.StringVar(&configFilename, "config", usr.HomeDir+"/.pganalyze_collector.conf", "Specify alternative path for config file.")
	flag.StringVar(&pidFilename, "pidfile", "", "Specifies a path that a pidfile should be written to. (default is no pidfile being written)")
	flag.Parse()

	globalCollectionOpts := collectionOpts{
		submitCollectedData:      true,
		testRun:                  testRun,
		collectPostgresRelations: !noPostgresRelations,
		collectPostgresSettings:  !noPostgresSettings,
		collectPostgresLocks:     !noPostgresLocks,
		collectPostgresFunctions: !noPostgresFunctions,
		collectPostgresBloat:     !noPostgresBloat,
		collectPostgresViews:     !noPostgresViews,
		collectLogs:              !noLogs,
		collectExplain:           !noExplain,
		collectSystemInformation: !noSystemInformation,
		diffStatements:           diffStatements,
	}

	if dryRun {
		globalCollectionOpts.submitCollectedData = false
		globalCollectionOpts.testRun = true
	} else {
		// Check some cases we can't support from a pganalyze perspective right now
		if noPostgresRelations {
			logger.PrintError("Error: You can only disable relation data collection for dry test runs (the API can't accept the snapshot otherwise)")
			return
		}
	}

	if testRun {
		globalCollectionOpts.collectorApplicationName = "pganalyze_test_run"
	} else {
		globalCollectionOpts.collectorApplicationName = "pganalyze"
	}

	if pidFilename != "" {
		pid := os.Getpid()
		err := ioutil.WriteFile(pidFilename, []byte(strconv.Itoa(pid)), 0644)
		if err != nil {
			logger.PrintError("Could not write pidfile to \"%s\" as requested, exiting.", pidFilename)
			return
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	wg := sync.WaitGroup{}

ReadConfigAndRun:
	stop := run(wg, globalCollectionOpts, logger, configFilename)
	if stop == nil {
		return
	}

	// Block here until we get any of the registered signals
	s := <-sigs

	// Stop the scheduled runs
	stop <- true

	if s == syscall.SIGHUP {
		logger.PrintInfo("Reloading configuration...")
		goto ReadConfigAndRun
	}

	signal.Stop(sigs)

	logger.PrintInfo("Exiting...")
	wg.Wait()
}
