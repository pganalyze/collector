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

	"github.com/golang/protobuf/proto"
	flag "github.com/ogier/pflag"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/scheduler"
	"github.com/pganalyze/collector/snapshot"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	"database/sql"

	_ "github.com/lib/pq" // Enable database package to use Postgres
)

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
	collectedAt time.Time
	statements  map[statementStatsKey]state.PostgresStatement
}

type database struct {
	config           config.DatabaseConfig
	connection       *sql.DB
	prevState        statsState
	requestedSslMode string
}

func collectStatistics(db database, collectionOpts collectionOpts, logger *util.Logger) (newState statsState, err error) {
	var s state.State
	var explainInputs []state.PostgresExplainInput

	postgresVersion, err := postgres.GetPostgresVersion(logger, db.connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	/*stats.Postgres = &snapshot.SnapshotPostgres{}
	stats.Postgres.Version = &postgresVersion*/

	if postgresVersion.Numeric < state.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required.", postgresVersion.Short)
		return
	}

	s.Backends, err = postgres.GetBackends(logger, db.connection, postgresVersion)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_activity")
		return
	}

	fmt.Printf("%+v\n", s.Backends)

	statements, err := postgres.GetStatements(logger, db.connection, postgresVersion)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_statements")
		return
	}

	if postgresVersion.Numeric >= state.PostgresVersion94 {
		var diffedStatements []state.PostgresStatement

		newState.statements = make(map[statementStatsKey]state.PostgresStatement)

		// Iterate through all statements, and diff them based on the previous state
		for _, statement := range statements {
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

		fmt.Printf("New state size: %d", len(newState.statements))

		var queryInformations []*snapshot.QueryInformation

		for _, statement := range diffedStatements {
			var queryInformation snapshot.QueryInformation

			queryInformation.NormalizedQuery = statement.Query

			queryInformations = append(queryInformations, &queryInformation)
		}

		//s.QueryInformations = queryInformations
		//stats.Opts.StatementStatsAreDiffed = true
	}

	if collectionOpts.collectPostgresRelations {
		s.Relations, err = postgres.GetRelations(db.connection, postgresVersion, collectionOpts.collectPostgresBloat)
		if err != nil {
			logger.PrintError("Error collecting schema information")
			return
		}
	}

	if collectionOpts.collectPostgresSettings {
		s.Settings, err = postgres.GetSettings(db.connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	if collectionOpts.collectPostgresFunctions {
		s.Functions, err = postgres.GetFunctions(db.connection, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting stored procedures")
			return
		}
	}

	if collectionOpts.collectSystemInformation {
		systemState := system.GetSystemState(db.config, logger)
		s.System = &systemState
	}

	if collectionOpts.collectLogs {
		s.Logs, explainInputs = system.GetLogLines(db.config)

		if collectionOpts.collectExplain {
			s.Explains = postgres.RunExplain(db.connection, explainInputs)
		}
	}

	// FIXME: Need to transform state into snapshot
	statsProto, err := proto.Marshal(&snapshot.Snapshot{})
	if err != nil {
		logger.PrintError("Error marshaling statistics")
		return
	}

	if !collectionOpts.submitCollectedData {
		statsReRead := &snapshot.Snapshot{}
		if err := proto.Unmarshal(statsProto, statsReRead); err != nil {
			log.Fatalln("Failed to re-read stats:", err)
		}

		var out bytes.Buffer
		statsJSON, _ := json.Marshal(statsReRead)
		json.Indent(&out, statsJSON, "", "\t")
		logger.PrintInfo("Dry run - data that would have been sent will be output on stdout:\n")
		fmt.Print(out.String())
		return
	}

	var compressedJSON bytes.Buffer
	w := zlib.NewWriter(&compressedJSON)
	w.Write(statsProto)
	w.Close()

	requestURL := db.config.APIBaseURL + "/v1/snapshots"

	if collectionOpts.testRun {
		requestURL = db.config.APIBaseURL + "/v1/snapshots/test"
	}

	data := url.Values{
		"data":            {compressedJSON.String()},
		"data_compressor": {"zlib"},
		"api_key":         {db.config.APIKey},
		"submitter":       {"pganalyze-collector 0.9.0rc7"},
		"no_reset":        {"true"},
		"query_source":    {"pg_stat_statements"},
		"collected_at":    {fmt.Sprintf("%d", time.Now().Unix())},
	}

	encodedData := data.Encode()

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(encodedData))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	logger.PrintVerbose("Successfully prepared request - size of request body: %.4f MB", float64(len(encodedData))/1024.0/1024.0)

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

	connection.QueryRow(postgres.QueryMarkerSQL + "SELECT COUNT(*) FROM pg_stat_activity WHERE application_name = '" + globalCollectionOpts.collectorApplicationName + "'").Scan(&connectionCount)

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

	schedulerGroups, err := scheduler.GetSchedulerGroups()
	if err != nil {
		logger.PrintError("Error: Could not get scheduler groups, awaiting SIGHUP or process kill")
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
	/*if globalCollectionOpts.testRun {
		collectAllDatabases(databases, globalCollectionOpts, logger)
		return nil
	}FIXME*/

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
