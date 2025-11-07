package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/juju/syslog"

	flag "github.com/ogier/pflag"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/runner"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	_ "github.com/lib/pq" // Enable database package to use Postgres
)

const defaultConfigFile = "/etc/pganalyze-collector.conf"
const defaultStateFile = "/var/lib/pganalyze-collector/state"

func main() {
	var showVersion bool
	var dryRun bool
	var dryRunLogs bool
	var analyzeLogfile string
	var analyzeLogfilePrefix string
	var analyzeLogfileTz string
	var analyzeDebugClassifications string
	var filterLogFile string
	var filterLogSecret string
	var debugLogs bool
	var discoverLogLocation bool
	var testRun bool
	var testRunLogs bool
	var testExplain bool
	var testSection string
	var generateStatsHelperSql string
	var generateHelperExplainAnalyzeSql string
	var generateHelperExplainAnalyzeRole string
	var forceStateUpdate bool
	var configFilename string
	var stateFilename string
	var pidFilename string
	var noPostgresSettings, noPostgresLocks bool
	var noPostgresRelations, noLogs, noExplain, noSystemInformation bool
	var writeHeapProfile bool
	var testRunAndTrace bool
	var logToSyslog bool
	var logToJSON bool
	var logNoTimestamps bool
	var reload bool
	var noReload bool
	var benchmark bool
	var veryVerbose bool
	var requireWebsocket bool

	logFlags := log.LstdFlags
	logger := &util.Logger{}

	flag.BoolVarP(&showVersion, "version", "", false, "Shows current version of the collector and exits")
	flag.BoolVarP(&testRun, "test", "t", false, "Tests data collection (including logs), submits it to the server, and reloads the collector daemon (disable with --no-reload)")
	flag.BoolVar(&testRunLogs, "test-logs", false, "Tests whether log collection works (does not test privilege dropping for local log collection, use --test for that)")
	flag.BoolVar(&testExplain, "test-explain", false, "Tests whether EXPLAIN collection works by issuing a dummy query (ensure log collection works first)")
	flag.StringVar(&testSection, "test-section", "", "Tests a particular section of the config file, i.e. a specific server, and ignores all other config sections")
	flag.StringVar(&generateStatsHelperSql, "generate-stats-helper-sql", "", "Generates a SQL script for the given server (name of section in the config file, or \"default\" for env variables), that can be run with \"psql -f\" for installing the collector stats helpers on all configured databases")
	flag.StringVar(&generateHelperExplainAnalyzeSql, "generate-explain-analyze-helper-sql", "", "Generates a SQL script for the given server (name of section in the config file, or \"default\" for env variables), that can be run with \"psql -f\" for installing the collector pganalyze.explain_analyze helper on all configured databases")
	flag.StringVar(&generateHelperExplainAnalyzeRole, "generate-explain-analyze-helper-role", "pganalyze_explain", "Sets owner role of the pganalyze.explain_analyze helper function, defaults to \"pganalyze_explain\"")
	flag.BoolVar(&reload, "reload", false, "Reloads the collector daemon that's running on the host")
	flag.BoolVar(&noReload, "no-reload", false, "Disables automatic config reloading during a test run")
	flag.BoolVarP(&logger.Verbose, "verbose", "v", false, "Outputs additional debugging information, use this if you're encountering errors or other problems")
	flag.BoolVar(&veryVerbose, "very-verbose", false, "Enable very verbose logging (will also enable verbose logging)")
	flag.BoolVarP(&logger.Quiet, "quiet", "q", false, "Only outputs error messages to the logs and hides informational and warning messages")
	flag.BoolVar(&logToSyslog, "syslog", false, "Write all log output to syslog instead of stderr (disabled by default)")
	flag.BoolVar(&logToJSON, "json-logs", false, "Write all log output to stderr as newline delimited json (disabled by default, ignored if --syslog is set)")
	flag.BoolVar(&logNoTimestamps, "no-log-timestamps", false, "Disable timestamps in the log output (automatically done when syslog is enabled)")
	flag.BoolVar(&dryRun, "dry-run", false, "Print JSON data that would get sent to web service (without actually sending) and exit afterwards")
	flag.BoolVar(&dryRunLogs, "dry-run-logs", false, "Print JSON data for log snapshot (without actually sending) and exit afterwards")
	flag.StringVar(&analyzeLogfile, "analyze-logfile", "", "Analyzes the content of the given log file and returns debug output about it")
	flag.StringVar(&analyzeLogfilePrefix, "analyze-logfile-prefix", "", "The log_line_prefix to use with --analyze-logfile")
	flag.StringVar(&analyzeLogfileTz, "analyze-logfile-tz", "", "The log_timezone to use with --analyze-logfile (default: UTC)")
	flag.StringVar(&analyzeDebugClassifications, "analyze-debug-classifications", "", "When used with --analyze-logfile, print detailed information about given classifications (can be comma-separated list of integer classifications, or keyword 'all')")
	flag.StringVar(&filterLogFile, "filter-logfile", "", "Test command that filters all known secrets in the logfile according to the filter-log-secret option")
	flag.StringVar(&filterLogSecret, "filter-log-secret", "all", "Sets the type of secrets filtered by the filter-logfile test command (default: all)")
	flag.BoolVar(&debugLogs, "debug-logs", false, "Outputs all log analysis that would be sent, doesn't send any other data. For some providers, it also outputs incoming logs from the source (use for debugging only)")
	flag.BoolVar(&discoverLogLocation, "discover-log-location", false, "Tries to automatically discover the location of the Postgres log directory, to support configuring the 'db_log_location' setting")
	flag.BoolVar(&forceStateUpdate, "force-state-update", false, "Updates the state file even if other options would have prevented it (intended to be used together with --dry-run for debugging)")
	flag.BoolVar(&noPostgresRelations, "no-postgres-relations", false, "Don't collect any Postgres relation information (not recommended)")
	flag.BoolVar(&noPostgresSettings, "no-postgres-settings", false, "Don't collect Postgres configuration settings")
	flag.BoolVar(&noPostgresLocks, "no-postgres-locks", false, "Don't collect Postgres lock information")
	flag.BoolVar(&noLogs, "no-logs", false, "Don't collect log data")
	flag.BoolVar(&noExplain, "no-explain", false, "Don't automatically EXPLAIN slow queries logged in the logfile")
	flag.BoolVar(&noSystemInformation, "no-system-information", false, "Don't collect OS level performance data")
	flag.BoolVar(&writeHeapProfile, "write-heap-profile", false, "Write a Go memory heap profile to ~/pganalyze_collector.mprof when SIGHUP is received (disabled by default, only useful for debugging)")
	flag.BoolVar(&testRunAndTrace, "trace", false, "Write a Go trace file to ~/pganalyze_collector.trace for a single test run (only useful for debugging)")
	flag.StringVar(&configFilename, "config", defaultConfigFile, "Specify alternative path for config file")
	flag.StringVar(&stateFilename, "statefile", defaultStateFile, "Specify alternative path for state file")
	flag.StringVar(&pidFilename, "pidfile", "", "Specifies a path that a pidfile should be written to (default is no pidfile being written)")
	flag.BoolVar(&benchmark, "benchmark", false, "Runs collector in benchmark mode (skip submitting the statistics to the server)")
	flag.BoolVar(&requireWebsocket, "require-websocket", true, "Require WebSocket connection to the pganalyze server")
	flag.Parse()

	// Automatically reload the configuration after a successful test run.
	reload = reload || (testRun && !noReload)

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
		logger.UseJSON = logToJSON
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

	if testRunLogs || testRunAndTrace || testExplain || generateStatsHelperSql != "" || generateHelperExplainAnalyzeSql != "" {
		testRun = true
	}

	if veryVerbose {
		logger.Verbose = true
	}

	opts := state.CollectionOpts{
		StartedAt:                        time.Now(),
		SubmitCollectedData:              !benchmark && true,
		TestRun:                          testRun,
		TestRunLogs:                      testRunLogs || dryRunLogs,
		TestExplain:                      testExplain,
		TestSection:                      testSection,
		GenerateStatsHelperSql:           generateStatsHelperSql,
		GenerateExplainAnalyzeHelperSql:  generateHelperExplainAnalyzeSql,
		GenerateExplainAnalyzeHelperRole: generateHelperExplainAnalyzeRole,
		DebugLogs:                        debugLogs,
		DiscoverLogLocation:              discoverLogLocation,
		CollectPostgresRelations:         !noPostgresRelations,
		CollectPostgresSettings:          !noPostgresSettings,
		CollectPostgresLocks:             !noPostgresLocks,
		CollectLogs:                      !noLogs,
		CollectExplain:                   !noExplain,
		CollectSystemInformation:         !noSystemInformation,
		StateFilename:                    stateFilename,
		WriteStateUpdate:                 (!dryRun && !dryRunLogs && !testRun) || forceStateUpdate,
		ForceEmptyGrant:                  dryRun || dryRunLogs || testRunLogs || benchmark,
		OutputAsJson:                     !benchmark,
		VeryVerbose:                      veryVerbose,
		RequireWebsocket:                 requireWebsocket,
	}

	if reload && !testRun {
		Reload(logger)
		return
	}

	if dryRun || dryRunLogs {
		opts.SubmitCollectedData = false
		opts.TestRun = true
	}

	if opts.TestRun || opts.TestRunLogs ||
		opts.DebugLogs || opts.DiscoverLogLocation {
		opts.CollectorApplicationName = "pganalyze_test_run"
	} else {
		opts.CollectorApplicationName = "pganalyze_collector"
	}

	if analyzeLogfile != "" {
		contentBytes, err := os.ReadFile(analyzeLogfile)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			return
		}
		logReader := logs.NewMaybeHerokuLogReader(strings.NewReader(string(contentBytes)))
		server := state.MakeServer(config.ServerConfig{}, false)
		tz, err := time.LoadLocation(analyzeLogfileTz)
		if err != nil {
			fmt.Printf("ERROR: could not read time zone: %s\n", err)
			return
		}
		if analyzeLogfilePrefix == "" {
			fmt.Println("ERROR: must specify log_line_prefix used to generate logfile with --analyze-logfile-prefix")
			return
		}
		server.LogParser = logs.NewLogParser(analyzeLogfilePrefix, tz)

		logLines, samples := logs.ParseAndAnalyzeBuffer(logReader, time.Time{}, server)
		logs.PrintDebugInfo(logLines, samples)
		if analyzeDebugClassifications != "" {
			classifications := strings.Split(analyzeDebugClassifications, ",")
			classMap := make(map[pganalyze_collector.LogLineInformation_LogClassification]bool)
			for _, classification := range classifications {
				if classification == "all" {
					// we represent "all" as an empty map
					continue
				}
				classVal, err := strconv.ParseInt(classification, 10, 32)
				if err != nil {
					fmt.Printf("ERROR: invalid classification: %s\n", err)
				}
				classInt := int32(classVal)
				classMap[pganalyze_collector.LogLineInformation_LogClassification(classInt)] = true
			}
			logs.PrintDebugLogLines(logLines, classMap)
		}
		return
	}

	if filterLogFile != "" {
		contentBytes, err := os.ReadFile(filterLogFile)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			return
		}
		logReader := logs.NewMaybeHerokuLogReader(strings.NewReader(string(contentBytes)))
		logLines, _ := logs.ParseAndAnalyzeBuffer(logReader, time.Time{}, state.MakeServer(config.ServerConfig{}, false))
		logs.ReplaceSecrets(logLines, state.ParseFilterLogSecret(filterLogSecret))
		output := ""
		for _, logLine := range logLines {
			output += logLine.Content
		}
		fmt.Printf("%s", output)
		return
	}

	if pidFilename != "" {
		pid := os.Getpid()
		err := os.WriteFile(pidFilename, []byte(strconv.Itoa(pid)), 0644)
		if err != nil {
			logger.PrintError("Could not write pidfile to \"%s\" as requested, exiting.", pidFilename)
			return
		}
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
		defer f.Close()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

ReadConfigAndRun:
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	exitCode := 0
	keepRunning, testRunSuccess, writeStateFile, shutdown := runner.Run(ctx, &wg, opts, logger, configFilename)

	if keepRunning {
		// Block here until we get any of the registered signals
		s := <-sigs

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
			shutdown()
			cancel()
			wg.Wait()
			writeStateFile()
			goto ReadConfigAndRun
		}

		logger.PrintInfo("Exiting...")
	} else {
		// The run function started some work (e.g. a test command), wait for that to finish before exiting
		done := make(chan struct{})
		go func() {
			defer close(done)
			wg.Wait()
		}()
	DoneOrSignal:
		for {
			select {
			case success := <-testRunSuccess:
				if reload {
					if success {
						Reload(logger)
					} else {
						logger.PrintError("Error: Reload requested, but ignoring since configuration errors are present")
						exitCode = 1
					}
				} else if !success {
					exitCode = 1
				}
				break DoneOrSignal
			case s := <-sigs:
				if s == syscall.SIGINT || s == syscall.SIGTERM {
					logger.PrintError("Interrupt")
					break DoneOrSignal
				}
			}
		}
	}

	shutdown()
	cancel()
	wg.Wait()

	signal.Stop(sigs)

	if testRunAndTrace {
		trace.Stop()
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func Reload(logger *util.Logger) {
	if util.IsHeroku() {
		return
	}
	pid, err := util.Reload()
	if err != nil {
		logger.PrintError("Error: Failed to reload collector: %s\n", err)
		os.Exit(1)
	}
	logger.PrintInfo("Successfully reloaded pganalyze collector (PID %d)\n", pid)
	os.Exit(0)
}
