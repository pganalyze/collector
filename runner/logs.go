package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/input/system/azure"
	"github.com/pganalyze/collector/input/system/google_cloudsql"
	"github.com/pganalyze/collector/input/system/heroku"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/input/system/tembo"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/logs/querysample"
	"github.com/pganalyze/collector/logs/stream"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

const LogDownloadInterval time.Duration = 30 * time.Second
const LogStreamingInterval time.Duration = 10 * time.Second

// SetupLogCollection - Starts streaming or scheduled downloads for logs of the specified servers
func SetupLogCollection(ctx context.Context, wg *sync.WaitGroup, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger, hasAnyHeroku bool, hasAnyGoogleCloudSQL bool, hasAnyAzureDatabase bool, hasAnyTembo bool) {
	var hasAnyLogDownloads bool
	var hasAnyLogTails bool

	for _, server := range servers {
		if server.Config.DisableLogs || server.Pause.Load() {
			continue
		}
		if server.Config.LogLocation != "" || server.Config.LogDockerTail != "" || server.Config.LogSyslogServer != "" || server.Config.LogOtelServer != "" {
			hasAnyLogTails = true
		} else if server.Config.SupportsLogDownload() {
			hasAnyLogDownloads = true
		}
	}

	var parsedLogStream chan state.ParsedLogStreamItem
	if hasAnyLogTails || hasAnyHeroku || hasAnyGoogleCloudSQL || hasAnyAzureDatabase || hasAnyTembo {
		parsedLogStream = setupLogStreamer(ctx, wg, opts, logger, servers, nil, stream.LogTestNone)
	}
	if hasAnyLogTails {
		selfhosted.SetupLogTails(ctx, wg, opts, logger, servers, parsedLogStream)
	}
	if hasAnyHeroku {
		heroku.SetupHttpHandlerLogs(ctx, wg, opts, logger, servers, parsedLogStream)
		for _, server := range servers {
			EmitTestLogMsg(ctx, server, opts, logger)
		}
	}
	if hasAnyGoogleCloudSQL {
		google_cloudsql.SetupLogSubscriber(ctx, wg, opts, logger, servers, parsedLogStream)
	}
	if hasAnyAzureDatabase {
		azure.SetupLogSubscriber(ctx, wg, opts, logger, servers, parsedLogStream)
	}
	if hasAnyTembo {
		tembo.SetupWebsocketHandlerLogs(ctx, wg, logger, servers, opts, parsedLogStream)
	}

	if hasAnyLogDownloads {
		setupLogDownloadForAllServers(ctx, wg, opts, logger, servers)
	}
}

func setupLogDownloadForAllServers(ctx context.Context, wg *sync.WaitGroup, opts state.CollectionOpts, logger *util.Logger, servers []*state.Server) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(LogDownloadInterval)

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				var innerWg sync.WaitGroup

				if !opts.CollectLogs {
					return
				}

				for _, server := range servers {
					grant := server.Grant.Load()
					if server.Config.DisableLogs || (grant.Valid && !grant.Config.EnableLogs) {
						continue
					}

					if !server.Config.SupportsLogDownload() {
						continue
					}

					innerWg.Add(1)
					go downloadLogsForServerWithLocksAndCallbacks(ctx, &innerWg, server, opts, logger)
				}

				innerWg.Wait()
			}
		}
	}()
}

func downloadLogsForServerWithLocksAndCallbacks(ctx context.Context, wg *sync.WaitGroup, server *state.Server, opts state.CollectionOpts, logger *util.Logger) {
	defer wg.Done()
	prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

	server.CollectionStatusMutex.Lock()
	if server.CollectionStatus.LogSnapshotDisabled {
		server.LogStateMutex.Lock()
		server.LogPrevState = state.PersistedLogState{}
		server.LogStateMutex.Unlock()
		server.CollectionStatusMutex.Unlock()
		return
	}
	server.CollectionStatusMutex.Unlock()

	server.LogStateMutex.Lock()
	newLogState, success, err := downloadLogsForServer(ctx, server, opts, prefixedLogger)
	if err != nil {
		server.LogStateMutex.Unlock()
		printLogDownloadError(server, err, prefixedLogger)
		if server.Config.ErrorCallback != "" {
			go runCompletionCallback("error", server.Config.ErrorCallback, server.Config.SectionName, "logs", err, prefixedLogger)
		}
	} else {
		server.LogPrevState = newLogState
		server.LogStateMutex.Unlock()
		if success && server.Config.SuccessCallback != "" {
			go runCompletionCallback("success", server.Config.SuccessCallback, server.Config.SectionName, "logs", nil, prefixedLogger)
		}
	}
}

func downloadLogsForServer(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (state.PersistedLogState, bool, error) {
	grant, err := output.GetGrant(ctx, server, opts, logger)
	if err != nil || !grant.Valid {
		return server.LogPrevState, false, err
	}
	transientLogState := state.TransientLogState{CollectedAt: time.Now()}

	var newLogState state.PersistedLogState
	newLogState, transientLogState.LogFiles, transientLogState.QuerySamples, err = system.DownloadLogFiles(ctx, server, opts, logger)
	if err != nil {
		return newLogState, false, errors.Wrap(err, "could not collect logs")
	}

	err = postprocessAndSendLogs(ctx, server, opts, logger, transientLogState, grant)
	if err != nil {
		return newLogState, false, err
	}
	return newLogState, true, nil
}

func findServerByIdentifier(servers []*state.Server, identifier config.ServerIdentifier) *state.Server {
	for _, s := range servers {
		if s.Config.Identifier == identifier {
			return s
		}
	}
	return nil
}

func setupLogStreamer(ctx context.Context, wg *sync.WaitGroup, opts state.CollectionOpts, logger *util.Logger, servers []*state.Server, logTestSucceeded chan<- bool, logTestFunc func(s *state.Server, lf state.LogFile, lt chan<- bool)) chan state.ParsedLogStreamItem {
	parsedLogStream := make(chan state.ParsedLogStreamItem, state.LogStreamBufferLen)

	wg.Add(1)
	go func() {
		defer wg.Done()
		logLinesByServer := make(map[config.ServerIdentifier][]state.LogLine)

		ticker := time.NewTicker(LogStreamingInterval)
		if opts.TestRun {
			ticker = time.NewTicker(1 * time.Second)
		}

		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case t := <-ticker.C:
				for identifier := range logLinesByServer {
					if len(logLinesByServer[identifier]) == 0 {
						continue
					}

					server := findServerByIdentifier(servers, identifier)
					if server == nil {
						// This should never happen, but in case it does, avoid memory leaks for data that can never be sent
						logger.PrintError("ERROR: Could not locate server entry for identifier \"%s\", discarding log lines", identifier)
						delete(logLinesByServer, identifier)
						continue
					}
					prefixedLogger := logger.WithPrefix(server.Config.SectionName)
					logLinesByServer[identifier] = processLogStream(ctx, server, logLinesByServer[identifier], t, opts, prefixedLogger, logTestSucceeded, logTestFunc)
				}
			case in, ok := <-parsedLogStream:
				var err error
				if !ok {
					return
				}

				in.LogLine.CollectedAt = time.Now()
				in.LogLine.UUID, err = uuid.NewV7()
				if err != nil {
					logger.PrintError("Could not generate log line UUID: %s", err)
					continue
				}
				logLinesByServer[in.Identifier] = append(logLinesByServer[in.Identifier], in.LogLine)
			}
		}
	}()

	return parsedLogStream
}

func processLogStream(ctx context.Context, server *state.Server, logLines []state.LogLine, now time.Time, opts state.CollectionOpts, logger *util.Logger, logTestSucceeded chan<- bool, logTestFunc func(s *state.Server, lf state.LogFile, lt chan<- bool)) []state.LogLine {
	server.CollectionStatusMutex.Lock()
	if server.CollectionStatus.LogSnapshotDisabled {
		server.CollectionStatusMutex.Unlock()
		return []state.LogLine{}
	}
	server.CollectionStatusMutex.Unlock()

	transientLogState, logFile, tooFreshLogLines, err := stream.AnalyzeStreamInGroups(logLines, now, server, logger)
	if err != nil {
		logger.PrintError("%s", err)
		return tooFreshLogLines
	}

	transientLogState.LogFiles = []state.LogFile{logFile}

	// Nothing to send, so just skip getting the grant and other work
	if len(logFile.LogLines) == 0 && len(transientLogState.QuerySamples) == 0 {
		return tooFreshLogLines
	}

	if opts.TestRun {
		grant := server.Grant.Load()
		if !grant.Valid || grant.Config.EnableLogs {
			server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectLogs)
			logTestFunc(server, logFile, logTestSucceeded)
		} else {
			server.SelfTest.MarkCollectionAspectError(state.CollectionAspectLogs, "Log Insights not available on this plan")
			server.SelfTest.HintCollectionAspect(state.CollectionAspectLogs, "You may need to upgrade, see %s", selftest.URLPrinter.Sprint("https://pganalyze.com/pricing"))
			logger.PrintError("  Failed - Log Insights feature not available on this pganalyze plan. You may need to upgrade, see https://pganalyze.com/pricing")
		}
		return tooFreshLogLines
	}

	grant, err := output.GetGrant(ctx, server, opts, logger)
	if err != nil {
		// Note we intentionally discard log lines here (and in the other
		// error case below), because the HTTP client already retries to work
		// around temporary failues, and otherwise we would keep processing
		// more and more lines in error scenarios
		logger.PrintError("Log sending error (discarding lines): %s", err)
		return tooFreshLogLines
	}
	if !grant.Valid {
		return tooFreshLogLines // Don't retry (e.g. because this feature is not available)
	}

	err = postprocessAndSendLogs(ctx, server, opts, logger, transientLogState, grant)
	if err != nil {
		logger.PrintError("Log sending error (discarding lines): %s", err)
		return tooFreshLogLines
	}

	return tooFreshLogLines
}

func postprocessAndSendLogs(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger, transientLogState state.TransientLogState, grant state.Grant) (err error) {
	if server.Config.EnableLogExplain && len(transientLogState.QuerySamples) != 0 {
		transientLogState.QuerySamples = postgres.RunExplain(ctx, server, transientLogState.QuerySamples, opts, logger)
	}

	if server.Config.FilterQuerySample == "all" {
		transientLogState.QuerySamples = []state.PostgresQuerySample{}
	} else if server.Config.FilterQuerySample == "normalize" {
		for idx, sample := range transientLogState.QuerySamples {
			// Ensure we always normalize the query text (when sample normalization is on), even if EXPLAIN errors out
			sample.Query = util.NormalizeQuery(sample.Query, "unparseable", -1)
			for pIdx := range sample.Parameters {
				sample.Parameters[pIdx] = null.StringFrom("<removed>")
			}
			if sample.ExplainOutputText != "" {
				sample.ExplainOutputText = ""
				sample.ExplainError = "EXPLAIN normalize failed: auto_explain format is not JSON - not supported (discarding EXPLAIN)"
			}
			if sample.ExplainOutputJSON != nil {
				// remove parameters as part of normalization
				sample.ExplainOutputJSON.QueryParameters = ""
				sample.ExplainOutputJSON, err = querysample.NormalizeExplainJSON(sample.ExplainOutputJSON)
				if err != nil {
					sample.ExplainOutputJSON = nil
					sample.ExplainError = fmt.Sprintf("EXPLAIN normalize failed: %s", err)
				}
			}
			transientLogState.QuerySamples[idx] = sample
		}
	} else {
		// Do nothing if filter_query_sample = none (we just take the query samples as they are generated)
	}

	// Export query samples as traces, if OpenTelemetry endpoint is configured
	if server.Config.OTelTracingProvider != nil && len(transientLogState.QuerySamples) != 0 {
		querysample.ExportQuerySamplesAsTraceSpans(ctx, server, logger, grant, transientLogState.QuerySamples)
	}

	for idx := range transientLogState.LogFiles {
		// The actual filtering (aka masking of secrets) is done later in
		// EncryptAndUploadLogfiles, based on this setting
		transientLogState.LogFiles[idx].FilterLogSecret = state.ParseFilterLogSecret(server.Config.FilterLogSecret)
	}

	logsExist := false
	for idx := range transientLogState.LogFiles {
		logFile := &transientLogState.LogFiles[idx]
		if len(logFile.LogLines) > 0 {
			logsExist = true
			logFile.ByteSize = int64(logFile.LogLines[len(logFile.LogLines)-1].ByteEnd)
		}
		if len(logFile.FilterLogSecret) > 0 {
			logs.ReplaceSecrets(logFile.LogLines, logFile.FilterLogSecret)
		}
	}

	if opts.DebugLogs {
		logger.PrintInfo("Would have sent log state:\n")
		for _, logFile := range transientLogState.LogFiles {
			logs.PrintDebugInfo(logFile.LogLines, transientLogState.QuerySamples)
		}
		return nil
	}

	if logsExist {
		err = output.UploadAndSendLogs(ctx, server, grant, opts, logger, transientLogState)
		if err != nil {
			server.SelfTest.MarkCollectionAspectError(state.CollectionAspectLogs, "error sending logs: %s", err)
			return errors.Wrap(err, "failed to upload/send logs")
		}
	}

	return nil
}

// TestLogsForAllServers - Test log download/tailing
func TestLogsForAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) (hasFailedServers bool, hasSuccessfulLocalServers bool) {
	if !opts.TestRun {
		return
	}

	for _, server := range servers {
		if server.Config.DisableLogs {
			continue
		}

		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)
		if server.CollectionStatus.LogSnapshotDisabled {
			prefixedLogger.PrintWarning("WARNING - Configuration issue: %s", server.CollectionStatus.LogSnapshotDisabledReason)
			prefixedLogger.PrintWarning("  Log collection will be disabled for this server")
			continue
		}

		logLinePrefix, err := getLogLinePrefix(ctx, server, opts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("ERROR - Could not check log_line_prefix for server: %s", err)
			hasFailedServers = true
			continue
		} else if server.Config.SystemType == "heroku" && logLinePrefix == logs.LogPrefixHerokuHobbyTier {
			prefixedLogger.PrintWarning("WARNING - Detected log_line_prefix indicates Heroku Postgres Hobby tier, which has no log output support")
			continue
		}

		if server.Config.LogSyslogServer != "" {
			prefixedLogger.PrintInfo("Skipping test for log collection (syslog server) - verify log snapshots are sent in collector logs")
			continue
		}

		ctx, cancel := context.WithCancel(context.Background())
		wg := sync.WaitGroup{}
		success := false

		if server.Config.LogLocation != "" {
			if testLocalLogTail(ctx, &wg, server, opts, prefixedLogger) {
				hasSuccessfulLocalServers = true
				success = true
			} else {
				success = false
			}
		} else if server.Config.SupportsLogDownload() {
			success = testLogDownload(ctx, &wg, server, opts, prefixedLogger)
		} else if server.Config.AzureEventhubNamespace != "" && server.Config.AzureEventhubName != "" {
			if server.Config.AzureDbServerName == "" {
				prefixedLogger.PrintError("ERROR - Detected Azure Event Hub setup but azure_db_server_name is not set")
			} else {
				success = testAzureLogStream(ctx, &wg, server, opts, prefixedLogger)
			}
		} else if server.Config.GcpCloudSQLInstanceID != "" && server.Config.GcpPubsubSubscription != "" {
			success = testGoogleCloudsqlLogStream(ctx, &wg, server, opts, prefixedLogger)
		} else if server.Config.LogOtelServer != "" {
			success = testOtelLog(ctx, &wg, server, opts, prefixedLogger)
		}

		if !success {
			hasFailedServers = true
		}

		cancel()
	}

	return
}

func getLogLinePrefix(ctx context.Context, server *state.Server, opts state.CollectionOpts, prefixedLogger *util.Logger) (string, error) {
	db, err := postgres.EstablishConnection(ctx, server, prefixedLogger, opts, "")
	if err != nil {
		return "", fmt.Errorf("Could not connect to database to retrieve log_line_prefix: %s", err)
	}
	defer db.Close()

	return postgres.GetPostgresSetting(ctx, db, "log_line_prefix")
}

func testLocalLogTail(ctx context.Context, wg *sync.WaitGroup, server *state.Server, opts state.CollectionOpts, logger *util.Logger) bool {
	logger.PrintInfo("Testing log collection (local)...")

	logTestSucceeded := make(chan bool, 1)
	parsedLogStream := setupLogStreamer(ctx, wg, opts, logger, []*state.Server{server}, logTestSucceeded, stream.LogTestCollectorIdentify)

	err := selfhosted.SetupLogTailForServer(ctx, wg, opts, logger, server, parsedLogStream)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectLogs, "error tailing logs for server: %s", err)
		logger.PrintError("ERROR - Could not tail logs for server: %s", err)
		return false
	}

	EmitTestLogMsg(ctx, server, opts, logger)

	select {
	case <-logTestSucceeded:
		break
	case <-time.After(10 * time.Second):
		logger.PrintError("ERROR - Local log tail timed out after 10 seconds - did not find expected log event in stream")
		return false
	}

	logger.PrintInfo("  Local log test successful")
	return true
}

func testLogDownload(ctx context.Context, wg *sync.WaitGroup, server *state.Server, opts state.CollectionOpts, prefixedLogger *util.Logger) bool {
	prefixedLogger.PrintInfo("Testing log download...")
	_, _, err := downloadLogsForServer(ctx, server, opts, prefixedLogger)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectLogs, err.Error())
		printLogDownloadError(server, err, prefixedLogger)
		return false
	}

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectLogs)

	prefixedLogger.PrintInfo("  Log test successful")
	return true
}

func testAzureLogStream(ctx context.Context, wg *sync.WaitGroup, server *state.Server, opts state.CollectionOpts, logger *util.Logger) bool {
	logger.PrintInfo("Testing log collection (Azure Database)...")

	logTestSucceeded := make(chan bool, 1)
	parsedLogStream := setupLogStreamer(ctx, wg, opts, logger, []*state.Server{server}, logTestSucceeded, stream.LogTestAnyEvent)

	err := azure.SetupLogSubscriber(ctx, wg, opts, logger, []*state.Server{server}, parsedLogStream)
	if err != nil {
		logger.PrintError("ERROR - Could not get logs through Azure Event Hub: %s", err)
		return false
	}

	EmitTestLogMsg(ctx, server, opts, logger)

	select {
	case <-logTestSucceeded:
		break
	case <-time.After(10 * time.Second):
		logger.PrintError("ERROR - Azure Event Hub log tail timed out after 10 seconds - did not receive any log events")
		logger.PrintInfo("HINT - This error may be a false positive if the collector is also running in the background and consumes the same Azure Event Hub stream")
		return false
	}

	logger.PrintInfo("  Log test successful")
	return true
}

func testGoogleCloudsqlLogStream(ctx context.Context, wg *sync.WaitGroup, server *state.Server, opts state.CollectionOpts, logger *util.Logger) bool {
	logger.PrintInfo("Testing log collection (Google Cloud SQL)...")

	logTestSucceeded := make(chan bool, 1)
	parsedLogStream := setupLogStreamer(ctx, wg, opts, logger, []*state.Server{server}, logTestSucceeded, stream.LogTestCollectorIdentify)

	err := google_cloudsql.SetupLogSubscriber(ctx, wg, opts, logger, []*state.Server{server}, parsedLogStream)
	if err != nil {
		logger.PrintError("ERROR - Could not get logs through Google Cloud Pub/Sub: %s", err)
		return false
	}

	EmitTestLogMsg(ctx, server, opts, logger)

	select {
	case <-logTestSucceeded:
		break
	case <-time.After(10 * time.Second):
		logger.PrintError("ERROR - Google Cloud Pub/Sub log tail timed out after 10 seconds - did not find expected log event in stream")
		logger.PrintInfo("HINT - This error may be a false positive if the collector is also running in the background and consumes the same Google Cloud Pub/Sub stream")
		return false
	}

	logger.PrintInfo("  Log test successful")
	return true
}

func testOtelLog(ctx context.Context, wg *sync.WaitGroup, server *state.Server, opts state.CollectionOpts, logger *util.Logger) bool {
	logger.PrintInfo("Testing log collection (OpenTelemetry Log receiving)...")

	logTestSucceeded := make(chan bool, 1)
	parsedLogStream := setupLogStreamer(ctx, wg, opts, logger, []*state.Server{server}, logTestSucceeded, stream.LogTestCollectorIdentify)

	selfhosted.SetupOtelHandlerForServers(ctx, wg, opts, logger, []*state.Server{server}, parsedLogStream)

	EmitTestLogMsg(ctx, server, opts, logger)

	select {
	case <-logTestSucceeded:
		break
	case <-time.After(10 * time.Second):
		logger.PrintError("ERROR - OpenTelemetry log tail timed out after 10 seconds - did not find expected log event in stream")
		logger.PrintInfo("HINT - This error may be a false positive if the collector is also running in the background and receiving logs")
		return false
	}

	logger.PrintInfo("  Log test successful")
	return true
}

func printLogDownloadError(server *state.Server, err error, prefixedLogger *util.Logger) {
	prefixedLogger.PrintError("ERROR - Could not download logs: %s", err)
	msg := err.Error()
	if server.Config.SystemType == "amazon_rds" && strings.Contains(msg, "NoCredentialProviders") {
		prefixedLogger.PrintInfo("HINT - This may occur if you have not assigned an IAM role to the collector EC2 instance, and have not provided AWS credentials through another method")
	}
}
