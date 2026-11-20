package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"runtime/pprof"
	"runtime/trace"
	"slices"
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
	"github.com/pganalyze/collector/util/awsutil"

	_ "github.com/lib/pq" // Enable database package to use Postgres
)

const defaultConfigFile = "/etc/pganalyze-collector.conf"
const defaultStateFile = "/var/lib/pganalyze-collector/state"

// Time allotted for a snapshot to complete when a shutdown signal comes in.
const collectionStopTimeout = 30 * time.Second

func main() {
	var showVersion bool
	var dryRun bool
	var dryRunLogs bool
	var analyzeLogfile string
	var logfilePrefix string
	var logfileTz string
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
	var noPostgresRelations, noLogs, noSystemInformation bool
	var writeHeapProfile bool
	var writeTrace bool
	var logToSyslog bool
	var logToJSON bool
	var logNoTimestamps bool
	var reload bool
	var noReload bool
	var benchmark bool
	var veryVerbose bool

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
	flag.BoolVar(&logToJSON, "json-logs", false, "Write all log output to stderr as newline delimited json (disabled by default)")
	flag.BoolVar(&logNoTimestamps, "no-log-timestamps", false, "Disable timestamps in the log output (automatically done when syslog is enabled)")
	flag.BoolVar(&dryRun, "dry-run", false, "Print JSON data that would get sent to web service (without actually sending) and exit afterwards")
	flag.BoolVar(&dryRunLogs, "dry-run-logs", false, "Print JSON data for log snapshot (without actually sending) and exit afterwards")
	flag.StringVar(&analyzeLogfile, "analyze-logfile", "", "Analyzes the content of the given log file and returns debug output about it")
	flag.StringVar(&logfilePrefix, "logfile-prefix", "", "The log_line_prefix to use with --analyze-logfile or --filter-logfile")
	flag.StringVar(&logfileTz, "logfile-tz", "", "The log_timezone to use with --analyze-logfile or --filter-logfile (default: UTC)")
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
	flag.BoolVar(&noSystemInformation, "no-system-information", false, "Don't collect OS level performance data")
	flag.BoolVar(&writeHeapProfile, "write-heap-profile", false, "Write a Go memory heap profile to ~/pganalyze_collector.mprof when SIGHUP is received (disabled by default, only useful for debugging)")
	flag.BoolVar(&writeTrace, "trace", false, "Write a Go trace file to ~/pganalyze_collector.trace during a test or dry run (only useful for debugging)")
	flag.StringVar(&configFilename, "config", defaultConfigFile, "Specify alternative path for config file")
	flag.StringVar(&stateFilename, "statefile", defaultStateFile, "Specify alternative path for state file")
	flag.StringVar(&pidFilename, "pidfile", "", "Specifies a path that a pidfile should be written to (default is no pidfile being written)")
	flag.BoolVar(&benchmark, "benchmark", false, "Runs collector in benchmark mode (skip submitting the statistics to the server)")
	flag.Parse()

	// Automatically reload the configuration after a successful test run.
	reload = reload || (testRun && !noReload)

	if showVersion {
		fmt.Printf("%s\n", util.CollectorVersion)
		return
	}

	// Sanity-check flag combinations
	givenFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { givenFlags[f.Name] = true })
	if err := checkFlagCombination(givenFlags); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(2)
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

	if testRunLogs || testExplain || generateStatsHelperSql != "" || generateHelperExplainAnalyzeSql != "" {
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
		CollectSystemInformation:         !noSystemInformation,
		StateFilename:                    stateFilename,
		WriteStateUpdate:                 (!dryRun && !dryRunLogs && !testRun) || forceStateUpdate,
		ForceEmptyGrant:                  dryRun || dryRunLogs || testRunLogs || benchmark,
		OutputAsJson:                     !benchmark,
		VeryVerbose:                      veryVerbose,
	}

	if dryRun || dryRunLogs {
		opts.SubmitCollectedData = false
		opts.TestRun = true
	}

	if reload && !opts.TestRun {
		Reload(logger)
		return
	}

	if opts.TestRun || opts.TestRunLogs ||
		opts.DebugLogs || opts.DiscoverLogLocation {
		opts.CollectorApplicationName = "pganalyze_test_run"
	} else {
		opts.CollectorApplicationName = "pganalyze_collector"
	}

	if analyzeLogfile != "" {
		var classifications map[pganalyze_collector.LogLineInformation_LogClassification]bool
		if analyzeDebugClassifications != "" {
			var err error
			classifications, err = parseLogClassifications(analyzeDebugClassifications)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
				os.Exit(1)
			}
		}
		contentBytes, err := os.ReadFile(analyzeLogfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
		logReader := logs.NewMaybeHerokuLogReader(strings.NewReader(string(contentBytes)))
		server, err := makeStubLogServer(logfilePrefix, logfileTz)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}

		logLines, samples := logs.ParseAndAnalyzeBuffer(logReader, time.Time{}, server, opts, logger)
		logs.PrintDebugInfo(logLines, samples)
		if analyzeDebugClassifications != "" {
			logs.PrintDebugLogLines(logLines, classifications)
		}
		return
	}

	if filterLogFile != "" {
		contentBytes, err := os.ReadFile(filterLogFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
		logReader := logs.NewMaybeHerokuLogReader(strings.NewReader(string(contentBytes)))
		server, err := makeStubLogServer(logfilePrefix, logfileTz)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
		logLines, _ := logs.ParseAndAnalyzeBuffer(logReader, time.Time{}, server, opts, logger)
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

	if writeTrace {
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

	if !testRun {
		util.PruneTempFiles(logger)
	}

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
			if !util.WaitWithTimeout(&wg, collectionStopTimeout) {
				// Can't be sure of collection state at this point: exit to be safe.
				logger.PrintError("Collection did not stop within %s, exiting instead of reloading", collectionStopTimeout)
				os.Exit(1)
			}
			writeStateFile()
			awsutil.ClearAccounts()
			opts.StartedAt = time.Now() // Signal to pganalyze API that collector essentially restarted
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
	if !util.WaitWithTimeout(&wg, collectionStopTimeout) {
		logger.PrintError("Collection did not stop within %s, exiting anyway", collectionStopTimeout)
	}

	signal.Stop(sigs)

	if writeTrace {
		trace.Stop()
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// Each of these flags selects a separate code path; only allow one
var modeFlags = []string{
	"test",
	"test-logs",
	"test-explain",
	"dry-run",
	"dry-run-logs",
	"generate-stats-helper-sql",
	"generate-explain-analyze-helper-sql",
	"analyze-logfile",
	"filter-logfile",
	"debug-logs",
	"discover-log-location",
}

const daemonMode = ""

var testRunModes = []string{"test", "test-logs", "test-explain", "dry-run", "dry-run-logs"}
var logfileModes = []string{"analyze-logfile", "filter-logfile"}
var collectionModes = []string{daemonMode, "test", "dry-run"}

// Flags that only have an effect in some modes and the modes they apply to
var modifierFlags = []struct {
	flag  string
	modes []string
}{
	{"reload", slices.Concat([]string{"reload"}, testRunModes)},
	// --test is the only mode that reloads by default
	{"no-reload", []string{"test"}},
	{"trace", testRunModes},
	{"test-section", testRunModes},
	{"benchmark", []string{daemonMode}},
	{"write-heap-profile", []string{daemonMode, "debug-logs"}},
	// Only these modes collect full snapshots without updating the state file
	{"force-state-update", []string{"test", "dry-run"}},
	{"filter-log-secret", []string{"filter-logfile"}},
	{"analyze-debug-classifications", []string{"analyze-logfile"}},
	{"logfile-prefix", logfileModes},
	{"logfile-tz", logfileModes},
	{"generate-explain-analyze-helper-role", []string{"generate-explain-analyze-helper-sql"}},
	{"no-postgres-relations", collectionModes},
	{"no-postgres-settings", collectionModes},
	{"no-postgres-locks", collectionModes},
	{"no-system-information", collectionModes},
	{"no-logs", collectionModes},
	{"pidfile", []string{daemonMode, "test", "test-logs", "test-explain", "dry-run", "dry-run-logs", "generate-stats-helper-sql", "generate-explain-analyze-helper-sql", "debug-logs", "discover-log-location"}},
}

// Pairs of flags where one would silently override the other.
var conflictingFlags = [][2]string{
	// Syslog output has its own format
	{"syslog", "json-logs"},
	// Inherently opposites
	{"reload", "no-reload"},
	{"quiet", "verbose"},
	{"quiet", "very-verbose"},
}

// checkFlagCombination returns an error on invalid combinations
func checkFlagCombination(givenFlags map[string]bool) error {
	var modes []string
	for _, f := range modeFlags {
		if givenFlags[f] {
			modes = append(modes, f)
		}
	}
	if len(modes) > 1 {
		return fmt.Errorf("%s can't be combined", joinList(flagNames(modes), "and"))
	}
	for _, pair := range conflictingFlags {
		if givenFlags[pair[0]] && givenFlags[pair[1]] {
			return fmt.Errorf("%s can't be combined", joinList(flagNames(pair[:]), "and"))
		}
	}
	mode := daemonMode
	switch {
	case len(modes) == 1:
		mode = modes[0]
	case givenFlags["reload"]:
		mode = "reload"
	}
	for _, m := range modifierFlags {
		if !givenFlags[m.flag] || slices.Contains(m.modes, mode) {
			continue
		}
		applies := fmt.Sprintf("--%s only applies to %s", m.flag, describeModes(m.modes, m.flag))
		if mode == daemonMode {
			return errors.New(applies)
		}
		return fmt.Errorf("--%s can't be combined with --%s (%s)", m.flag, mode, applies)
	}
	return nil
}

// describeModes lists the modes a modifier flag applies to, omitting the
// implicit mode set by the flag itself (if applicable).
func describeModes(modes []string, flag string) string {
	var names []string
	for _, m := range modes {
		switch m {
		case flag:
		case daemonMode:
			names = append(names, "the collector daemon")
		default:
			names = append(names, "--"+m)
		}
	}
	return joinList(names, "or")
}

func flagNames(names []string) []string {
	flags := make([]string, len(names))
	for i, name := range names {
		flags[i] = "--" + name
	}
	return flags
}

// joinList formats items as "a and b" or "a, b or c".
func joinList(items []string, conjunction string) string {
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + conjunction + " " + items[len(items)-1]
}

// parseLogClassifications parses a comma-separated list of log classification
// numbers, or "all", for logs.PrintDebugLogLines. "all" returns an empty map,
// which PrintDebugLogLines treats as every classification.
func parseLogClassifications(list string) (map[pganalyze_collector.LogLineInformation_LogClassification]bool, error) {
	classifications := make(map[pganalyze_collector.LogLineInformation_LogClassification]bool)
	if list == "all" {
		return classifications, nil
	}
	for _, entry := range strings.Split(list, ",") {
		if entry == "all" {
			return nil, errors.New("\"all\" can't be combined with other classifications")
		}
		value, err := strconv.ParseInt(entry, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid classification %q: must be an integer or \"all\"", entry)
		}
		if _, ok := pganalyze_collector.LogLineInformation_LogClassification_name[int32(value)]; !ok {
			return nil, fmt.Errorf("unknown classification %d", value)
		}
		classifications[pganalyze_collector.LogLineInformation_LogClassification(value)] = true
	}
	return classifications, nil
}

func makeStubLogServer(prefix string, tzName string) (*state.Server, error) {
	if prefix == "" {
		return nil, errors.New("must specify log_line_prefix used to generate logfile with --logfile-prefix")
	}
	tz, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("could not read time zone: %s", err)
	}
	server := state.MakeServer(config.ServerConfig{}, false)
	server.LogParser = logs.NewLogParser(prefix, tz, false)
	return server, nil
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
