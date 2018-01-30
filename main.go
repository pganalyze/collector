package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"os/user"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/juju/syslog"

	flag "github.com/ogier/pflag"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/system/heroku"
	"github.com/pganalyze/collector/input/system/logs"
	"github.com/pganalyze/collector/runner"
	"github.com/pganalyze/collector/scheduler"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	_ "github.com/lib/pq" // Enable database package to use Postgres
)

func run(wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, configFilename string) (bool, chan<- bool, chan<- bool, chan<- bool, chan<- bool) {
	var servers []state.Server

	schedulerGroups, err := scheduler.GetSchedulerGroups()
	if err != nil {
		logger.PrintError("Error: Could not get scheduler groups")
		return false, nil, nil, nil, nil
	}

	conf, err := config.Read(logger, configFilename)
	if err != nil {
		logger.PrintError("Config Error: %s", err)
		return !globalCollectionOpts.TestRun, nil, nil, nil, nil
	}

	serverConfigs := conf.Servers

	for _, config := range serverConfigs {
		servers = append(servers, state.Server{Config: config})
	}

	runner.ReadStateFile(servers, globalCollectionOpts, logger)

	// We intentionally don't do a test-run in the normal mode, since we're fine with
	// a later SIGHUP that fixes the config (or a temporarily unreachable server at start)
	if globalCollectionOpts.TestRun {
		if globalCollectionOpts.TestReport != "" {
			runner.RunTestReport(servers, globalCollectionOpts, logger)
		} else if globalCollectionOpts.TestRunLogs {
			runner.CollectLogsFromAllServers(servers, globalCollectionOpts, logger)
		} else {
			runner.CollectAllServers(servers, globalCollectionOpts, logger)
		}
		return false, nil, nil, nil, nil
	}

	statsStop := schedulerGroups["stats"].Schedule(func() {
		wg.Add(1)
		runner.CollectAllServers(servers, globalCollectionOpts, logger)
		wg.Done()
	}, logger, "full snapshot of all servers")

	// Avoid even running the scheduler when we already know its not needed
	hasAnyLogsEnabled := false
	hasAnyReportsEnabled := false
	hasAnyActivityEnabled := false
	for _, server := range servers {
		if server.Config.EnableLogs {
			hasAnyLogsEnabled = true
		}
		if server.Config.EnableReports {
			hasAnyReportsEnabled = true
		}
		if server.Config.EnableActivity {
			hasAnyActivityEnabled = true
		}
	}

	var reportsStop chan<- bool
	if hasAnyReportsEnabled {
		reportsStop = schedulerGroups["reports"].Schedule(func() {
			wg.Add(1)
			runner.RunRequestedReports(servers, globalCollectionOpts, logger)
			wg.Done()
		}, logger, "requested reports for all servers")
	}

	var logsStop chan<- bool
	if hasAnyLogsEnabled {
		if conf.HerokuLogStream != nil {
			heroku.SetupLogReceiver(conf, servers, globalCollectionOpts, logger)
		} else {
			logsStop = schedulerGroups["logs"].Schedule(func() {
				wg.Add(1)
				runner.CollectLogsFromAllServers(servers, globalCollectionOpts, logger)
				wg.Done()
			}, logger, "log snapshot of all servers")
		}
	}

	var activityStop chan<- bool
	if hasAnyActivityEnabled {
		activityStop = schedulerGroups["activity"].Schedule(func() {
			wg.Add(1)
			runner.CollectActivityFromAllServers(servers, globalCollectionOpts, logger)
			wg.Done()
		}, logger, "activity snapshot of all servers")
	}

	return true, statsStop, reportsStop, logsStop, activityStop
}

const defaultConfigFile = "/etc/pganalyze-collector.conf"
const defaultStateFile = "/var/lib/pganalyze-collector/state"

func main() {
	var showVersion bool
	var dryRun bool
	var dryRunLogs bool
	var analyzeLogfile string
	var testRun bool
	var testReport string
	var forceStateUpdate bool
	var configFilename string
	var stateFilename string
	var pidFilename string
	var noPostgresSettings, noPostgresLocks, noPostgresFunctions, noPostgresBloat, noPostgresViews bool
	var noPostgresRelations, noLogs, noExplain, noSystemInformation, diffStatements bool
	var writeHeapProfile bool
	var testRunAndTrace bool
	var logToSyslog bool
	var logNoTimestamps bool
	var reloadRun bool

	logFlags := log.LstdFlags
	logger := &util.Logger{}

	flag.BoolVarP(&showVersion, "version", "", false, "Shows current version of the collector and exits")
	flag.BoolVarP(&testRun, "test", "t", false, "Tests whether we can successfully collect data, submits it to the server, and exits afterwards")
	flag.StringVar(&testReport, "test-report", "", "Tests a particular report and returns its output as JSON")
	flag.BoolVar(&reloadRun, "reload", false, "Reloads the collector daemon thats running on the host")
	flag.BoolVarP(&logger.Verbose, "verbose", "v", false, "Outputs additional debugging information, use this if you're encoutering errors or other problems")
	flag.BoolVar(&logToSyslog, "syslog", false, "Write all log output to syslog instead of stderr (disabled by default)")
	flag.BoolVar(&logNoTimestamps, "no-log-timestamps", false, "Disable timestamps in the log output (automatically done when syslog is enabled)")
	flag.BoolVar(&dryRun, "dry-run", false, "Print JSON data that would get sent to web service (without actually sending) and exit afterwards")
	flag.BoolVar(&dryRunLogs, "dry-run-logs", false, "Print JSON data for log snapshot (without actually sending) and exit afterwards")
	flag.StringVar(&analyzeLogfile, "analyze-logfile", "", "Analyzes the content of the given log file and returns debug output about it")
	flag.BoolVar(&forceStateUpdate, "force-state-update", false, "Updates the state file even if other options would have prevented it (intended to be used together with --dry-run for debugging)")
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
	flag.BoolVar(&writeHeapProfile, "write-heap-profile", false, "Write a Go memory heap profile to ~/pganalyze_collector.mprof when SIGHUP is received (disabled by default, only useful for debugging)")
	flag.BoolVar(&testRunAndTrace, "trace", false, "Write a Go trace file to ~/pganalyze_collector.trace for a single test run (only useful for debugging)")
	flag.StringVar(&configFilename, "config", defaultConfigFile, "Specify alternative path for config file")
	flag.StringVar(&stateFilename, "statefile", defaultStateFile, "Specify alternative path for state file")
	flag.StringVar(&pidFilename, "pidfile", "", "Specifies a path that a pidfile should be written to (default is no pidfile being written)")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s\n", util.CollectorVersion)
		return
	}

	if logNoTimestamps || logToSyslog {
		logFlags = 0
	}

	if logToSyslog {
		var err error
		logger.Destination, err = syslog.NewLogger(syslog.LOG_NOTICE|syslog.LOG_DAEMON, logFlags)
		if err != nil {
			panic(fmt.Errorf("Could not setup syslog as requested: %s", err))
		}
	} else {
		logger.Destination = log.New(os.Stderr, "", logFlags)
	}

	if configFilename == defaultConfigFile {
		_, err := os.Stat(configFilename)
		if os.IsNotExist(err) {
			// Fall back to the previous location of config files, to ease transitions
			usr, err := user.Current()
			if err == nil {
				configFilename = usr.HomeDir + "/.pganalyze_collector.conf"
			}
		}
	}

	if testReport != "" {
		testRun = true
	}

	if testRunAndTrace {
		testRun = true
	}

	globalCollectionOpts := state.CollectionOpts{
		SubmitCollectedData:      true,
		TestRun:                  testRun,
		TestReport:               testReport,
		TestRunLogs:              dryRunLogs,
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
		WriteStateUpdate:         (!dryRun && !dryRunLogs && !testRun) || forceStateUpdate,
		ForceEmptyGrant:          dryRun || dryRunLogs,
	}

	if reloadRun {
		util.Reload()
		return
	}

	if dryRun || dryRunLogs {
		globalCollectionOpts.SubmitCollectedData = false
		globalCollectionOpts.TestRun = true
	}

	if globalCollectionOpts.TestRun || globalCollectionOpts.TestReport != "" {
		globalCollectionOpts.CollectorApplicationName = "pganalyze_test_run"
	} else {
		globalCollectionOpts.CollectorApplicationName = "pganalyze_collector"
	}

	if analyzeLogfile != "" {
		content, err := ioutil.ReadFile(analyzeLogfile)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			return
		}
		logLines, samples, _ := logs.ParseAndAnalyzeBuffer(string(content), 0, time.Time{})
		logs.PrintDebugInfo(logLines, samples)
		return
	}

	if testRunAndTrace {
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		tracePath := usr.HomeDir + "/pganalyze_collector.trace"
		f, err := os.Create(tracePath)
		if err != nil {
			panic(err)
		}
		trace.Start(f)
		run(&sync.WaitGroup{}, globalCollectionOpts, logger, configFilename)
		trace.Stop()
		f.Close()
		return
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
	keepRunning, statsStop, reportsStop, logsStop, activityStop := run(&wg, globalCollectionOpts, logger, configFilename)
	if !keepRunning {
		return
	}

	// Block here until we get any of the registered signals
	s := <-sigs

	// Stop the scheduled runs
	if statsStop != nil {
		statsStop <- true
	}
	if reportsStop != nil {
		reportsStop <- true
	}
	if logsStop != nil {
		logsStop <- true
	}
	if activityStop != nil {
		activityStop <- true
	}

	if s == syscall.SIGHUP {
		if writeHeapProfile {
			usr, err := user.Current()
			if err == nil {
				mprofPath := usr.HomeDir + "/pganalyze_collector.mprof"
				f, err := os.Create(mprofPath)
				if err == nil {
					pprof.WriteHeapProfile(f)
					f.Close()
					logger.PrintInfo("Wrote memory heap profile to %s", mprofPath)
				}
			}
		}
		logger.PrintInfo("Reloading configuration...")
		wg.Wait()
		goto ReadConfigAndRun
	}

	signal.Stop(sigs)

	logger.PrintInfo("Exiting...")
	wg.Wait()
}
