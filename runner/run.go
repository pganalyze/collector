package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"sync"
	"syscall"

	"github.com/pkg/errors"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/scheduler"
	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/postgres/pgxv5"
)

func Run(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, configFilename string) (keepRunning bool, testRunSuccess chan bool, writeStateFile func(), shutdown func()) {
	var servers []*state.Server

	keepRunning = false
	writeStateFile = func() {}
	shutdown = func() {}
	var driverCleanup func() error

	scheduler, err := scheduler.GetScheduler()
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

		if cfg.DbUseIamAuth && cfg.SystemType == "google_cloudsql" && driverCleanup == nil {
			driverCleanup, err = pgxv5.RegisterDriver("cloudsql-postgres", cloudsqlconn.WithIAMAuthN())
			if err != nil {
				logger.PrintError("Failed to register cloudsql-postgres driver: %s", err)
				return
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
		if driverCleanup != nil {
			driverCleanup()
		}
	}

	// Avoid even running the scheduler when we already know its not needed
	hasAnyLogsEnabled := false
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
		servers = append(servers, state.MakeServer(config, globalCollectionOpts.TestRun))
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

	if globalCollectionOpts.GenerateStatsHelperSql != "" {
		wg.Add(1)
		testRunSuccess = make(chan bool)
		go func() {
			var matchingServer *state.Server
			for _, server := range servers {
				if globalCollectionOpts.GenerateStatsHelperSql == server.Config.SectionName {
					matchingServer = server
				}
			}
			if matchingServer == nil {
				fmt.Fprintf(os.Stderr, "ERROR - Specified configuration section name '%s' not known\n", globalCollectionOpts.GenerateStatsHelperSql)
				testRunSuccess <- false
			} else {
				output, err := GenerateStatsHelperSql(ctx, matchingServer, globalCollectionOpts, logger.WithPrefix(matchingServer.Config.SectionName))
				if err != nil {
					fmt.Fprintf(os.Stderr, "ERROR - %s\n", err)
					testRunSuccess <- false
				} else {
					fmt.Print(output)
					testRunSuccess <- true
				}
			}
			wg.Done()
		}()
		return
	}

	if globalCollectionOpts.GenerateExplainAnalyzeHelperSql != "" {
		wg.Add(1)
		testRunSuccess = make(chan bool)
		go func() {
			var matchingServer *state.Server
			for _, server := range servers {
				if globalCollectionOpts.GenerateExplainAnalyzeHelperSql == server.Config.SectionName {
					matchingServer = server
				}
			}
			if matchingServer == nil {
				fmt.Fprintf(os.Stderr, "ERROR - Specified configuration section name '%s' not known\n", globalCollectionOpts.GenerateExplainAnalyzeHelperSql)
				testRunSuccess <- false
			} else {
				output, err := GenerateExplainAnalyzeHelperSql(ctx, matchingServer, globalCollectionOpts, logger.WithPrefix(matchingServer.Config.SectionName))
				if err != nil {
					fmt.Fprintf(os.Stderr, "ERROR - %s\n", err)
					testRunSuccess <- false
				} else {
					fmt.Print(output)
					testRunSuccess <- true
				}
			}
			wg.Done()
		}()
		return
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
		// This channel is buffered so the function can exit (and mark the wait group as done)
		// without the caller consuming the channel, e.g. when the context gets canceled
		testRunSuccess = make(chan bool, 1)
		SetupWebsocketForAllServers(ctx, servers, globalCollectionOpts, logger)
		go func() {
			if globalCollectionOpts.TestExplain {
				success := true
				for _, server := range servers {
					prefixedLogger := logger.WithPrefix(server.Config.SectionName)
					err := EmitTestExplain(ctx, server, globalCollectionOpts, prefixedLogger)
					if err != nil {
						prefixedLogger.PrintError("Failed to run test explain: %s", err)
						success = false
					} else {
						prefixedLogger.PrintInfo("Emitted test explain; check pganalyze EXPLAIN Plans page for result")
					}
				}

				testRunSuccess <- success
			} else if globalCollectionOpts.TestRunLogs {
				success := doLogTest(ctx, servers, globalCollectionOpts, logger)
				testRunSuccess <- success
			} else {
				var allFullSuccessful bool
				var allActivitySuccessful bool
				allFullSuccessful = CollectAllServers(ctx, servers, globalCollectionOpts, logger)
				if ctx.Err() == nil {
					if hasAnyActivityEnabled {
						allActivitySuccessful = CollectActivityFromAllServers(ctx, servers, globalCollectionOpts, logger)
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

				if ctx.Err() == nil {
					selftest.PrintSummary(servers, logger.Verbose)
				}
				success := allFullSuccessful && allActivitySuccessful
				if success {
					fmt.Fprintln(os.Stderr, "Test successful")
					fmt.Fprintln(os.Stderr)
				}
				testRunSuccess <- success
			}
			wg.Done()
		}()
		return
	}

	if globalCollectionOpts.DebugLogs {
		SetupLogCollection(ctx, wg, servers, globalCollectionOpts, logger, hasAnyHeroku, hasAnyGoogleCloudSQL, hasAnyAzureDatabase, hasAnyTembo)

		// Keep running but only running log processing
		keepRunning = true
		return
	}

	if globalCollectionOpts.DiscoverLogLocation {
		selfhosted.DiscoverLogLocation(ctx, servers, globalCollectionOpts, logger)
		testRunSuccess = make(chan bool, 1)
		testRunSuccess <- true
		return
	}

	scheduler.TenMinute.Schedule(ctx, func(ctx context.Context) {
		wg.Add(1)
		CollectAllServers(ctx, servers, globalCollectionOpts, logger)
		wg.Done()
	}, logger, "full snapshot of all servers")

	if hasAnyLogsEnabled {
		SetupLogCollection(ctx, wg, servers, globalCollectionOpts, logger, hasAnyHeroku, hasAnyGoogleCloudSQL, hasAnyAzureDatabase, hasAnyTembo)
	} else if util.IsHeroku() {
		// Even if logs are deactivated, Heroku still requires us to have a functioning web server
		util.SetupHttpHandlerDummy()
	}

	if hasAnyActivityEnabled {
		scheduler.TenSecond.Schedule(ctx, func(ctx context.Context) {
			wg.Add(1)
			CollectActivityFromAllServers(ctx, servers, globalCollectionOpts, logger)
			wg.Done()
		}, logger, "activity snapshot of all servers")
	}

	// This captures stats every minute, except for minute 10 when full snapshot collection takes over
	scheduler.OneMinute.ScheduleSecondary(ctx, scheduler.TenMinute, func(ctx context.Context) {
		wg.Add(1)
		Gather1minStatsFromAllServers(ctx, servers, globalCollectionOpts, logger)
		wg.Done()
	}, logger, "high frequency statistics of all servers")

	SetupWebsocketForAllServers(ctx, servers, globalCollectionOpts, logger)
	SetupQueryRunnerForAllServers(ctx, servers, globalCollectionOpts, logger)

	keepRunning = true
	return
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
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectMonitoringDbConnection, err.Error())
		return errors.Wrap(err, "failed to connect to database")
	}
	defer conn.Close()
	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectMonitoringDbConnection)

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
		server.SelfTest.MarkCollectionSuspended("all monitoring suspended for this server: %s", collectionDisabledReason)
	} else if logsDisabled {
		logger.PrintInfo("Log collection suspended for this server: %s", logsDisabledReason)
	} else if logsIgnoreDuration {
		logger.PrintInfo("Log duration lines will be ignored for this server: %s", logsDisabledReason)
	} else if logsIgnoreStatement {
		logger.PrintInfo("Log statement lines will be ignored for this server: %s", logsDisabledReason)
	}

	logs.SyncLogParser(server, settings)
	parser := server.GetLogParser()
	if parser == nil {
		logger.PrintWarning("Could not initialize log parser for server")
	} else {
		prefixErr := parser.ValidatePrefix()
		if prefixErr != nil {
			logger.PrintWarning("Checking log_line_prefix: %s", prefixErr)
		}
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

func doLogTest(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) bool {
	// Initial test
	hasFailedServers, hasSuccessfulLocalServers := TestLogsForAllServers(ctx, servers, globalCollectionOpts, logger)

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
