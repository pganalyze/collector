package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"sync"
	"syscall"

	flag "github.com/ogier/pflag"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/runner"
	"github.com/pganalyze/collector/scheduler"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	_ "github.com/lib/pq" // Enable database package to use Postgres
)

func run(wg sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, configFilename string) chan<- bool {
	var servers []state.Server

	schedulerGroups, err := scheduler.GetSchedulerGroups()
	if err != nil {
		logger.PrintError("Error: Could not get scheduler groups, awaiting SIGHUP or process kill")
		return nil
	}

	serverConfigs, err := config.Read(configFilename)
	if err != nil {
		logger.PrintError("Error: Could not read configuration, awaiting SIGHUP or process kill")
		return nil
	}

	for _, config := range serverConfigs {
		server := state.Server{Config: config, RequestedSslMode: config.DbSslMode}

		// Go's lib/pq does not support sslmode properly, so we have to implement the "prefer" mode ourselves
		if server.RequestedSslMode == "prefer" {
			server.Config.DbSslMode = "require"
		}

		servers = append(servers, server)
	}

	runner.ReadStateFile(servers, globalCollectionOpts, logger)

	// We intentionally don't do a test-run in the normal mode, since we're fine with
	// a later SIGHUP that fixes the config (or a temporarily unreachable server at start)
	if globalCollectionOpts.TestRun {
		runner.CollectAllServers(servers, globalCollectionOpts, logger)
		return nil
	}

	stop := schedulerGroups["stats"].Schedule(func() {
		wg.Add(1)
		runner.CollectAllServers(servers, globalCollectionOpts, logger)
		wg.Done()
	}, logger, "collection of all databases")

	return stop
}

func main() {
	var dryRun bool
	var testRun bool
	var configFilename string
	var stateFilename string
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
	flag.StringVar(&stateFilename, "statefile", usr.HomeDir+"/.pganalyze_collector.state", "Specify alternative path for state file.")
	flag.StringVar(&pidFilename, "pidfile", "", "Specifies a path that a pidfile should be written to. (default is no pidfile being written)")
	flag.Parse()

	globalCollectionOpts := state.CollectionOpts{
		SubmitCollectedData:      true,
		TestRun:                  testRun,
		CollectPostgresRelations: !noPostgresRelations,
		CollectPostgresSettings:  !noPostgresSettings,
		CollectPostgresLocks:     !noPostgresLocks,
		CollectPostgresFunctions: !noPostgresFunctions,
		CollectPostgresBloat:     !noPostgresBloat,
		CollectPostgresViews:     !noPostgresViews,
		CollectLogs:              !noLogs,
		CollectExplain:           !noExplain,
		CollectSystemInformation: !noSystemInformation,
		DiffStatements:           diffStatements,
		StateFilename:            stateFilename,
		StatementTimeoutMs:       10000,
	}

	if dryRun {
		globalCollectionOpts.SubmitCollectedData = false
		globalCollectionOpts.TestRun = true
	} else {
		// Check some cases we can't support from a pganalyze perspective right now
		if noPostgresRelations {
			logger.PrintError("Error: You can only disable relation data collection for dry test runs (the API can't accept the snapshot otherwise)")
			return
		}
	}

	if testRun {
		globalCollectionOpts.CollectorApplicationName = "pganalyze_test_run"
	} else {
		globalCollectionOpts.CollectorApplicationName = "pganalyze_collector"
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
