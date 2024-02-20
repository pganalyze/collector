package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
	"github.com/pkg/errors"

	flag "github.com/ogier/pflag"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/runner"
	"github.com/pganalyze/collector/scheduler"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	_ "github.com/lib/pq" // Enable database package to use Postgres
)

func run(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, configFilename string) (keepRunning bool, testRunSuccess chan bool, writeStateFile func(), shutdown func()) {
	var servers []*state.Server

	keepRunning = false
	writeStateFile = func() {}
	shutdown = func() {}

	schedulerGroups, err := scheduler.GetSchedulerGroups()
	if err != nil {
		logger.PrintError("Error: Could not get scheduler groups")
		return
	}

	conf, err := config.Read(logger, configFilename)
	if err != nil {
		logger.PrintError("Config Error: %s", err)
		keepRunning = !globalCollectionOpts.TestRun && !globalCollectionOpts.DiscoverLogLocation
		return
	}

	for idx, cfg := range conf.Servers {
		prefixedLogger := logger.WithPrefix(cfg.SectionName)
		prefixedLogger.PrintVerbose("Identified as api_system_type: %s, api_system_scope: %s, api_system_id: %s", cfg.SystemType, cfg.SystemScope, cfg.SystemID)

		conf.Servers[idx].HTTPClient = config.CreateHTTPClient(cfg, prefixedLogger, false)
		conf.Servers[idx].HTTPClientWithRetry = config.CreateHTTPClient(cfg, prefixedLogger, true)
		if cfg.OtelExporterOtlpEndpoint != "" {
			conf.Servers[idx].OTelTracingProvider, conf.Servers[idx].OTelTracingProviderShutdownFunc, err = config.CreateOTelTracingProvider(ctx, cfg)
			logger.PrintVerbose("Initializing OpenTelemetry tracing provider with endpoint: %s", cfg.OtelExporterOtlpEndpoint)
			if err != nil {
				logger.PrintError("Failed to initialize OpenTelemetry tracing provider, disabling exports: %s", err)
			}
		}
	}

	shutdown = func() {
		for _, cfg := range conf.Servers {
			if cfg.OTelTracingProviderShutdownFunc == nil {
				continue
			}
			if err := cfg.OTelTracingProviderShutdownFunc(ctx); err != nil {
				logger.PrintError("Failed to shutdown OpenTelemetry tracing provider: %s", err)
			}
		}
	}

	// Avoid even running the scheduler when we already know its not needed
	hasAnyLogsEnabled := false
	hasAnyReportsEnabled := false
	hasAnyActivityEnabled := false
	hasAnyGoogleCloudSQL := false
	hasAnyAzureDatabase := false
	hasAnyHeroku := false
	hasAnyTembo := false

	serverConfigs := conf.Servers
	for _, config := range serverConfigs {
		if globalCollectionOpts.TestRun && globalCollectionOpts.TestSection != "" && globalCollectionOpts.TestSection != config.SectionName {
			continue
		}
		servers = append(servers, state.MakeServer(config))
		if config.EnableReports {
			hasAnyReportsEnabled = true
		}
		if !config.DisableLogs {
			hasAnyLogsEnabled = true
		}
		if !config.DisableActivity {
			hasAnyActivityEnabled = true
		}
		if config.SystemType == "azure_database" {
			hasAnyAzureDatabase = true
		}
		if config.SystemType == "google_cloudsql" {
			hasAnyGoogleCloudSQL = true
		}
		if config.SystemType == "heroku" {
			hasAnyHeroku = true
		}
		if config.SystemType == "tembo" {
			hasAnyTembo = true
		}
	}

	state.ReadStateFile(servers, globalCollectionOpts, logger)

	writeStateFile = func() {
		state.WriteStateFile(servers, globalCollectionOpts, logger)
	}

	if globalCollectionOpts.TestRun {
		logger.PrintInfo("Running collector test with %s", util.CollectorNameAndVersion)
	}

	checkAllInitialCollectionStatus(ctx, servers, globalCollectionOpts, logger)

	// We intentionally don't do a test-run in the normal mode, since we're fine with
	// a later SIGHUP that fixes the config (or a temporarily unreachable server at start)
	if globalCollectionOpts.TestRun {
		wg.Add(1)
		testRunSuccess = make(chan bool)
		go func() {
			if globalCollectionOpts.TestReport != "" {
				runner.RunTestReport(ctx, servers, globalCollectionOpts, logger)
				testRunSuccess <- true
			} else if globalCollectionOpts.TestExplain {
				success := true
				for _, server := range servers {
					prefixedLogger := logger.WithPrefix(server.Config.SectionName)
					err := runner.EmitTestExplain(ctx, server, globalCollectionOpts, prefixedLogger)
					if err != nil {
						prefixedLogger.PrintError("Failed to run test explain: %s", err)
						success = false
					}
				}
				testRunSuccess <- success
			} else if globalCollectionOpts.TestRunLogs {
				success := doLogTest(ctx, servers, globalCollectionOpts, logger)
				testRunSuccess <- success
			} else {
				var allFullSuccessful bool
				var allActivitySuccessful bool
				allFullSuccessful = runner.CollectAllServers(ctx, servers, globalCollectionOpts, logger)
				if ctx.Err() == nil {
					if hasAnyActivityEnabled {
						allActivitySuccessful = runner.CollectActivityFromAllServers(ctx, servers, globalCollectionOpts, logger)
					} else {
						allActivitySuccessful = true
					}
				}
				if hasAnyLogsEnabled && ctx.Err() == nil {
					// We intentionally don't fail for the regular test command if the log test fails, since you may not
					// have Log Insights enabled on your plan (which would fail the log test when getting the log grant).
					// In these situations we still want --test to be successful (i.e. issue a reload), but --test-logs
					// would fail (and not reload).
					doLogTest(ctx, servers, globalCollectionOpts, logger)
				}

				success := allFullSuccessful && allActivitySuccessful
				if success {
					// in a dry run, we will not actually have URLs; avoid this output in that case
					var hasURLs bool
					for _, server := range servers {
						if server.PGAnalyzeURL != "" {
							hasURLs = true
							break
						}
					}
					if hasURLs {
						fmt.Fprintln(os.Stderr)
						fmt.Fprintln(os.Stderr, "Test successful. View servers in pganalyze:")
						for _, server := range servers {
							if server.PGAnalyzeURL != "" {
								fmt.Fprintf(os.Stderr, " - [%s]: %s\n", server.Config.SectionName, server.PGAnalyzeURL)
							}
						}
						fmt.Fprintln(os.Stderr)
					}
				}
				testRunSuccess <- success
			}
			wg.Done()
		}()
		return
	}

	if globalCollectionOpts.DebugLogs {
		runner.SetupLogCollection(ctx, wg, servers, globalCollectionOpts, logger, hasAnyHeroku, hasAnyGoogleCloudSQL, hasAnyAzureDatabase, hasAnyTembo)

		// Keep running but only running log processing
		keepRunning = true
		return
	}

	if globalCollectionOpts.DiscoverLogLocation {
		selfhosted.DiscoverLogLocation(ctx, servers, globalCollectionOpts, logger)
		return
	}

	schedulerGroups["stats"].Schedule(ctx, func(ctx context.Context) {
		wg.Add(1)
		runner.CollectAllServers(ctx, servers, globalCollectionOpts, logger)
		wg.Done()
	}, logger, "full snapshot of all servers")

	if hasAnyReportsEnabled {
		schedulerGroups["reports"].Schedule(ctx, func(ctx context.Context) {
			wg.Add(1)
			runner.RunRequestedReports(ctx, servers, globalCollectionOpts, logger)
			wg.Done()
		}, logger, "requested reports for all servers")
	}

	if hasAnyLogsEnabled {
		runner.SetupLogCollection(ctx, wg, servers, globalCollectionOpts, logger, hasAnyHeroku, hasAnyGoogleCloudSQL, hasAnyAzureDatabase, hasAnyTembo)
	} else if os.Getenv("DYNO") != "" && os.Getenv("PORT") != "" {
		// Even if logs are deactivated, Heroku still requires us to have a functioning web server
		util.SetupHttpHandlerDummy()
	}

	if hasAnyActivityEnabled {
		schedulerGroups["activity"].Schedule(ctx, func(ctx context.Context) {
			wg.Add(1)
			runner.CollectActivityFromAllServers(ctx, servers, globalCollectionOpts, logger)
			wg.Done()
		}, logger, "activity snapshot of all servers")
	}

	schedulerGroups["query_stats"].ScheduleSecondary(ctx, func(ctx context.Context) {
		wg.Add(1)
		runner.GatherQueryStatsFromAllServers(ctx, servers, globalCollectionOpts, logger)
		wg.Done()
	}, logger, "high frequency query statistics of all servers", schedulerGroups["stats"])

	keepRunning = true
	return
}

const defaultConfigFile = "/etc/pganalyze-collector.conf"
const defaultStateFile = "/var/lib/pganalyze-collector/state"

func main() {
	var showVersion bool
	var dryRun bool
	var dryRunLogs bool
	var analyzeLogfile string
	var analyzeDebugClassifications string
	var filterLogFile string
	var filterLogSecret string
	var debugLogs bool
	var discoverLogLocation bool
	var testRun bool
	var testReport string
	var testRunLogs bool
	var testExplain bool
	var testSection string
	var forceStateUpdate bool
	var configFilename string
	var stateFilename string
	var pidFilename string
	var noPostgresSettings, noPostgresLocks, noPostgresFunctions, noPostgresBloat, noPostgresViews bool
	var noPostgresRelations, noLogs, noExplain, noSystemInformation bool
	var writeHeapProfile bool
	var testRunAndTrace bool
	var logToSyslog bool
	var logToJSON bool
	var logNoTimestamps bool
	var reloadRun bool
	var noReload bool
	var benchmark bool

	logFlags := log.LstdFlags
	logger := &util.Logger{}

	flag.BoolVarP(&showVersion, "version", "", false, "Shows current version of the collector and exits")
	flag.BoolVarP(&testRun, "test", "t", false, "Tests data collection (including logs), submits it to the server, and reloads the collector daemon (disable with --no-reload)")
	flag.StringVar(&testReport, "test-report", "", "Tests a particular report and returns its output as JSON")
	flag.BoolVar(&testRunLogs, "test-logs", false, "Tests whether log collection works (does not test privilege dropping for local log collection, use --test for that)")
	flag.BoolVar(&testExplain, "test-explain", false, "Tests whether EXPLAIN collection works by issuing a dummy query (ensure log collection works first)")
	flag.StringVar(&testSection, "test-section", "", "Tests a particular section of the config file, i.e. a specific server, and ignores all other config sections")
	flag.BoolVar(&reloadRun, "reload", false, "Reloads the collector daemon thats running on the host")
	flag.BoolVar(&noReload, "no-reload", false, "Disables automatic config reloading during a test run")
	flag.BoolVarP(&logger.Verbose, "verbose", "v", false, "Outputs additional debugging information, use this if you're encoutering errors or other problems")
	flag.BoolVarP(&logger.Quiet, "quiet", "q", false, "Only outputs error messages to the logs and hides informational and warning messages")
	flag.BoolVar(&logToSyslog, "syslog", false, "Write all log output to syslog instead of stderr (disabled by default)")
	flag.BoolVar(&logToJSON, "json-logs", false, "Write all log output to stderr as newline delimited json (disabled by default, ignored if --syslog is set)")
	flag.BoolVar(&logNoTimestamps, "no-log-timestamps", false, "Disable timestamps in the log output (automatically done when syslog is enabled)")
	flag.BoolVar(&dryRun, "dry-run", false, "Print JSON data that would get sent to web service (without actually sending) and exit afterwards")
	flag.BoolVar(&dryRunLogs, "dry-run-logs", false, "Print JSON data for log snapshot (without actually sending) and exit afterwards")
	flag.StringVar(&analyzeLogfile, "analyze-logfile", "", "Analyzes the content of the given log file and returns debug output about it")
	flag.StringVar(&analyzeDebugClassifications, "analyze-debug-classifications", "", "When used with --analyze-logfile, print detailed information about given classifications (can be comma-separated list of integer classifications, or keyword 'all')")
	flag.StringVar(&filterLogFile, "filter-logfile", "", "Test command that filters all known secrets in the logfile according to the filter-log-secret option")
	flag.StringVar(&filterLogSecret, "filter-log-secret", "all", "Sets the type of secrets filtered by the filter-logfile test command (default: all)")
	flag.BoolVar(&debugLogs, "debug-logs", false, "Outputs all log analysis that would be sent, doesn't send any other data (use for debugging only)")
	flag.BoolVar(&discoverLogLocation, "discover-log-location", false, "Tries to automatically discover the location of the Postgres log directory, to support configuring the 'db_log_location' setting")
	flag.BoolVar(&forceStateUpdate, "force-state-update", false, "Updates the state file even if other options would have prevented it (intended to be used together with --dry-run for debugging)")
	flag.BoolVar(&noPostgresRelations, "no-postgres-relations", false, "Don't collect any Postgres relation information (not recommended)")
	flag.BoolVar(&noPostgresSettings, "no-postgres-settings", false, "Don't collect Postgres configuration settings")
	flag.BoolVar(&noPostgresLocks, "no-postgres-locks", false, "Don't collect Postgres lock information")
	flag.BoolVar(&noPostgresFunctions, "no-postgres-functions", false, "Don't collect Postgres function/procedure information")
	flag.BoolVar(&noPostgresBloat, "no-postgres-bloat", false, "Don't collect Postgres table/index bloat statistics")
	flag.BoolVar(&noPostgresViews, "no-postgres-views", false, "Don't collect Postgres view/materialized view information (NOTE: This is not implemented right now - views are always collected)")
	flag.BoolVar(&noLogs, "no-logs", false, "Don't collect log data")
	flag.BoolVar(&noExplain, "no-explain", false, "Don't automatically EXPLAIN slow queries logged in the logfile")
	flag.BoolVar(&noSystemInformation, "no-system-information", false, "Don't collect OS level performance data")
	flag.BoolVar(&writeHeapProfile, "write-heap-profile", false, "Write a Go memory heap profile to ~/pganalyze_collector.mprof when SIGHUP is received (disabled by default, only useful for debugging)")
	flag.BoolVar(&testRunAndTrace, "trace", false, "Write a Go trace file to ~/pganalyze_collector.trace for a single test run (only useful for debugging)")
	flag.StringVar(&configFilename, "config", defaultConfigFile, "Specify alternative path for config file")
	flag.StringVar(&stateFilename, "statefile", defaultStateFile, "Specify alternative path for state file")
	flag.StringVar(&pidFilename, "pidfile", "", "Specifies a path that a pidfile should be written to (default is no pidfile being written)")
	flag.BoolVar(&benchmark, "benchmark", false, "Runs collector in benchmark mode (skip submitting the statistics to the server)")
	flag.Parse()

	// Automatically reload the configuration after a successful test run.
	if testRun && !noReload {
		reloadRun = true
	}

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

	if testReport != "" || testRunLogs || testRunAndTrace || testExplain {
		testRun = true
	}

	globalCollectionOpts := state.CollectionOpts{
		StartedAt:                time.Now(),
		SubmitCollectedData:      !benchmark && true,
		TestRun:                  testRun,
		TestReport:               testReport,
		TestRunLogs:              testRunLogs || dryRunLogs,
		TestExplain:              testExplain,
		TestSection:              testSection,
		DebugLogs:                debugLogs,
		DiscoverLogLocation:      discoverLogLocation,
		CollectPostgresRelations: !noPostgresRelations,
		CollectPostgresSettings:  !noPostgresSettings,
		CollectPostgresLocks:     !noPostgresLocks,
		CollectPostgresFunctions: !noPostgresFunctions,
		CollectPostgresBloat:     !noPostgresBloat,
		CollectPostgresViews:     !noPostgresViews,
		CollectLogs:              !noLogs,
		CollectExplain:           !noExplain,
		CollectSystemInformation: !noSystemInformation,
		StateFilename:            stateFilename,
		WriteStateUpdate:         (!dryRun && !dryRunLogs && !testRun) || forceStateUpdate,
		ForceEmptyGrant:          dryRun || dryRunLogs || benchmark,
		OutputAsJson:             !benchmark,
	}

	if reloadRun && !testRun {
		Reload(logger)
		return
	}

	if dryRun || dryRunLogs {
		globalCollectionOpts.SubmitCollectedData = false
		globalCollectionOpts.TestRun = true
	}

	if globalCollectionOpts.TestRun || globalCollectionOpts.TestReport != "" ||
		globalCollectionOpts.TestRunLogs || globalCollectionOpts.DebugLogs ||
		globalCollectionOpts.DiscoverLogLocation {
		globalCollectionOpts.CollectorApplicationName = "pganalyze_test_run"
	} else {
		globalCollectionOpts.CollectorApplicationName = "pganalyze_collector"
	}

	if analyzeLogfile != "" {
		contentBytes, err := ioutil.ReadFile(analyzeLogfile)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			return
		}
		content := string(contentBytes)
		reader := strings.NewReader(content)
		logReader := logs.NewMaybeHerokuLogReader(reader)
		logLines, samples := logs.ParseAndAnalyzeBuffer(logReader, time.Time{}, state.MakeServer(config.ServerConfig{}))
		logs.PrintDebugInfo(content, logLines, samples)
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
			logs.PrintDebugLogLines(content, logLines, classMap)
		}
		return
	}

	if filterLogFile != "" {
		contentBytes, err := ioutil.ReadFile(filterLogFile)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			return
		}
		content := string(contentBytes)
		reader := strings.NewReader(content)
		logReader := logs.NewMaybeHerokuLogReader(reader)
		logLines, _ := logs.ParseAndAnalyzeBuffer(logReader, time.Time{}, &state.Server{})
		output := logs.ReplaceSecrets(contentBytes, logLines, state.ParseFilterLogSecret(filterLogSecret))
		fmt.Printf("%s", output)
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
	keepRunning, testRunSuccess, writeStateFile, shutdown := run(ctx, &wg, globalCollectionOpts, logger, configFilename)

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
				if reloadRun {
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

func checkAllInitialCollectionStatus(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) {
	for _, server := range servers {
		var prefixedLogger = logger.WithPrefix(server.Config.SectionName)
		err := checkOneInitialCollectionStatus(ctx, server, opts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintVerbose("could not check initial collection status: %s", err)
		}
	}
}

func checkOneInitialCollectionStatus(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) error {
	conn, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}
	defer conn.Close()

	settings, err := postgres.GetSettings(ctx, conn)
	if err != nil {
		return err
	}

	if server.Config.DbDataDirectory == "" {
		// We don't need a mutex here, because we only do this once at startup
		server.Config.DbDataDirectory = postgres.GetDataDirectory(server, settings)
	}

	logsDisabled, logsIgnoreStatement, logsIgnoreDuration, logsDisabledReason := logs.ValidateLogCollectionConfig(server, settings)

	var isIgnoredReplica bool
	var collectionDisabledReason string
	if server.Config.SkipIfReplica {
		isIgnoredReplica, err = postgres.GetIsReplica(ctx, logger, conn)
		if err != nil {
			return err
		}
		if isIgnoredReplica {
			collectionDisabledReason = state.ErrReplicaCollectionDisabled.Error()
		}
	}
	if isIgnoredReplica {
		logger.PrintInfo("All monitoring suspended for this server: %s", collectionDisabledReason)
	} else if logsDisabled {
		logger.PrintInfo("Log collection suspended for this server: %s", logsDisabledReason)
	} else if logsIgnoreDuration {
		logger.PrintInfo("Log duration lines will be ignored for this server: %s", logsDisabledReason)
	} else if logsIgnoreStatement {
		logger.PrintInfo("Log statement lines will be ignored for this server: %s", logsDisabledReason)
	}

	server.SetLogTimezone(settings)
	if server.LogTimezone == nil {
		logger.PrintWarning("Could not determine log timezone for this server: %s")
	}

	server.CollectionStatusMutex.Lock()
	defer server.CollectionStatusMutex.Unlock()
	server.CollectionStatus = state.CollectionStatus{
		LogSnapshotDisabled:       logsDisabled,
		LogSnapshotDisabledReason: logsDisabledReason,
		CollectionDisabled:        isIgnoredReplica,
		CollectionDisabledReason:  collectionDisabledReason,
	}
	server.SetLogIgnoreFlags(logsIgnoreStatement, logsIgnoreDuration)

	return nil
}

func Reload(logger *util.Logger) {
	pid, err := util.Reload()
	if err != nil {
		logger.PrintError("Error: Failed to reload collector: %s\n", err)
		os.Exit(1)
	}
	logger.PrintInfo("Successfully reloaded pganalyze collector (PID %d)\n", pid)
	os.Exit(0)
}

func doLogTest(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) bool {
	// Initial test
	hasFailedServers, hasSuccessfulLocalServers := runner.TestLogsForAllServers(ctx, servers, globalCollectionOpts, logger)

	// Re-test using lower privileges
	if hasFailedServers {
		return false
	}
	if !hasSuccessfulLocalServers {
		return true
	}

	curUser, err := user.Current()
	if err != nil {
		logger.PrintError("Could not determine current user for privilege drop test")
		return false
	}
	if curUser.Name != "root" {
		// don't print anything here, since it would always be printed during the actual privilege drop run
		return true
	}

	pgaUser, err := user.Lookup("pganalyze")
	if err != nil {
		logger.PrintVerbose("Could not locate pganalyze user, skipping privilege drop test: %s", err)
		return true
	} else if curUser.Uid == pgaUser.Uid {
		logger.PrintVerbose("Current user is already pganalyze user, skipping privilege drop test")
		return true
	}

	uid, _ := strconv.ParseUint(pgaUser.Uid, 10, 32)
	gid, _ := strconv.ParseUint(pgaUser.Gid, 10, 32)
	groupIDStrs, _ := pgaUser.GroupIds()
	var groupIDs []uint32
	for _, groupIDStr := range groupIDStrs {
		groupID, _ := strconv.ParseUint(groupIDStr, 10, 32)
		groupIDs = append(groupIDs, uint32(groupID))
	}
	logger.PrintInfo("Re-running log test with reduced privileges of \"pganalyze\" user (uid = %d, gid = %d)", uid, gid)
	collectorBinaryPath, err := os.Executable()
	if err != nil {
		logger.PrintError("Could not run collector log test as \"pganalyze\" user due to missing executable: %s", err)
		return false
	}
	cmd := exec.Command(collectorBinaryPath, "--test-logs")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: groupIDs}
	err = cmd.Run()
	if err != nil {
		logger.PrintError("Could not run collector log test as \"pganalyze\" user: %s", err)
		return false
	}

	return true
}
