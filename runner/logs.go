package runner

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/input/system/azure"
	"github.com/pganalyze/collector/input/system/google_cloudsql"
	"github.com/pganalyze/collector/input/system/heroku"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/logs/querysample"
	"github.com/pganalyze/collector/logs/stream"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const LogDownloadInterval time.Duration = 30 * time.Second
const LogStreamingInterval time.Duration = 10 * time.Second

// SetupLogCollection - Starts streaming or scheduled downloads for logs of the specified servers
func SetupLogCollection(ctx context.Context, wg *sync.WaitGroup, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger, hasAnyHeroku bool, hasAnyGoogleCloudSQL bool, hasAnyAzureDatabase bool) {
	var hasAnyLogDownloads bool
	var hasAnyLogTails bool

	for _, server := range servers {
		if server.Config.DisableLogs {
			continue
		}
		if server.Config.LogLocation != "" || server.Config.LogDockerTail != "" || server.Config.LogSyslogServer != "" || server.Config.LogKubernetesPod != "" {
			hasAnyLogTails = true
		} else if server.Config.SupportsLogDownload() {
			hasAnyLogDownloads = true
		}
	}

	var parsedLogStream chan state.ParsedLogStreamItem
	if hasAnyLogTails || hasAnyHeroku || hasAnyGoogleCloudSQL || hasAnyAzureDatabase {
		parsedLogStream = setupLogStreamer(ctx, wg, globalCollectionOpts, logger, servers, nil, stream.LogTestNone)
	}
	if hasAnyLogTails {
		selfhosted.SetupLogTails(ctx, wg, globalCollectionOpts, logger, servers, parsedLogStream)
	}
	if hasAnyHeroku {
		heroku.SetupHttpHandlerLogs(ctx, wg, globalCollectionOpts, logger, servers, parsedLogStream)
		for _, server := range servers {
			EmitTestLogMsg(ctx, server, globalCollectionOpts, logger)
		}
	}
	if hasAnyGoogleCloudSQL {
		google_cloudsql.SetupLogSubscriber(ctx, wg, globalCollectionOpts, logger, servers, parsedLogStream)
	}
	if hasAnyAzureDatabase {
		azure.SetupLogSubscriber(ctx, wg, globalCollectionOpts, logger, servers, parsedLogStream)
	}

	if hasAnyLogDownloads {
		setupLogDownloadForAllServers(ctx, wg, globalCollectionOpts, logger, servers)
	}
}

func setupLogDownloadForAllServers(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server) {
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

				if !globalCollectionOpts.CollectLogs {
					return
				}

				for _, server := range servers {
					if server.Config.DisableLogs || (server.Grant.Valid && !server.Grant.Config.EnableLogs) {
						continue
					}

					if !server.Config.SupportsLogDownload() {
						continue
					}

					innerWg.Add(1)
					go downloadLogsForServerWithLocksAndCallbacks(ctx, &innerWg, server, globalCollectionOpts, logger)
				}

				innerWg.Wait()
			}
		}
	}()
}

func downloadLogsForServerWithLocksAndCallbacks(ctx context.Context, wg *sync.WaitGroup, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
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
	newLogState, success, err := downloadLogsForServer(ctx, server, globalCollectionOpts, prefixedLogger)
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

func downloadLogsForServer(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedLogState, bool, error) {
	grant, err := getLogsGrant(server, globalCollectionOpts, logger)
	if err != nil || !grant.Valid {
		return server.LogPrevState, false, err
	}

	transientLogState := state.TransientLogState{CollectedAt: time.Now()}
	defer transientLogState.Cleanup(logger)

	var newLogState state.PersistedLogState
	newLogState, transientLogState.LogFiles, transientLogState.QuerySamples, err = system.DownloadLogFiles(ctx, server, globalCollectionOpts, logger)
	if err != nil {
		return newLogState, false, errors.Wrap(err, "could not collect logs")
	}

	err = postprocessAndSendLogs(ctx, server, globalCollectionOpts, logger, transientLogState, grant)
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

func setupLogStreamer(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, logTestSucceeded chan<- bool, logTestFunc func(s *state.Server, lf state.LogFile, lt chan<- bool)) chan state.ParsedLogStreamItem {
	parsedLogStream := make(chan state.ParsedLogStreamItem, state.LogStreamBufferLen)

	wg.Add(1)
	go func() {
		defer wg.Done()
		logLinesByServer := make(map[config.ServerIdentifier][]state.LogLine)

		ticker := time.NewTicker(LogStreamingInterval)
		if globalCollectionOpts.TestRun {
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
					logLinesByServer[identifier] = processLogStream(ctx, server, logLinesByServer[identifier], t, globalCollectionOpts, prefixedLogger, logTestSucceeded, logTestFunc)
				}
			case in, ok := <-parsedLogStream:
				if !ok {
					return
				}

				in.LogLine.CollectedAt = time.Now()
				in.LogLine.UUID = uuid.NewV4()
				logLinesByServer[in.Identifier] = append(logLinesByServer[in.Identifier], in.LogLine)
			}
		}
	}()

	return parsedLogStream
}

func processLogStream(ctx context.Context, server *state.Server, logLines []state.LogLine, now time.Time, globalCollectionOpts state.CollectionOpts, logger *util.Logger, logTestSucceeded chan<- bool, logTestFunc func(s *state.Server, lf state.LogFile, lt chan<- bool)) []state.LogLine {
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
	defer transientLogState.Cleanup(logger)

	transientLogState.LogFiles = []state.LogFile{logFile}

	// Nothing to send, so just skip getting the grant and other work
	if len(logFile.LogLines) == 0 && len(transientLogState.QuerySamples) == 0 {
		return tooFreshLogLines
	}

	if globalCollectionOpts.TestRun {
		logTestFunc(server, logFile, logTestSucceeded)
		return tooFreshLogLines
	}

	grant, err := getLogsGrant(server, globalCollectionOpts, logger)
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

	err = postprocessAndSendLogs(ctx, server, globalCollectionOpts, logger, transientLogState, grant)
	if err != nil {
		logger.PrintError("Log sending error (discarding lines): %s", err)
		return tooFreshLogLines
	}

	return tooFreshLogLines
}

func getLogsGrant(server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (logGrant state.GrantLogs, err error) {
	logGrant, err = grant.GetLogsGrant(server, globalCollectionOpts, logger)
	if err != nil {
		return state.GrantLogs{Valid: false}, errors.Wrap(err, "could not get log grant")
	}

	if !logGrant.Valid {
		if globalCollectionOpts.TestRun {
			logger.PrintError("  Failed - Log Insights feature not available on this pganalyze plan, or log data limit exceeded. You may need to upgrade, see https://pganalyze.com/pricing")
		} else {
			logger.PrintVerbose("Skipping log data: Feature not available on this pganalyze plan, or log data limit exceeded")
		}
		return state.GrantLogs{Valid: false}, nil
	}

	return logGrant, nil
}

func postprocessAndSendLogs(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger, transientLogState state.TransientLogState, grant state.GrantLogs) (err error) {
	if server.Config.EnableLogExplain && len(transientLogState.QuerySamples) != 0 {
		transientLogState.QuerySamples = postgres.RunExplain(ctx, server, transientLogState.QuerySamples, globalCollectionOpts, logger)
	}

	if server.Config.FilterQuerySample == "all" {
		transientLogState.QuerySamples = []state.PostgresQuerySample{}
	} else if server.Config.FilterQuerySample == "normalize" {
		for idx, sample := range transientLogState.QuerySamples {
			// Ensure we always normalize the query text (when sample normalization is on), even if EXPLAIN errors out
			sample.Query = util.NormalizeQuery(sample.Query, "unparseable", -1)
			for pIdx, _ := range sample.Parameters {
				sample.Parameters[pIdx] = null.StringFrom("<removed>")
			}
			if sample.ExplainOutputText != "" {
				sample.ExplainOutputText = ""
				sample.ExplainError = "EXPLAIN normalize failed: auto_explain format is not JSON - not supported (discarding EXPLAIN)"
			}
			if sample.ExplainOutputJSON != nil {
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

	for idx := range transientLogState.LogFiles {
		// The actual filtering (aka masking of secrets) is done later in
		// EncryptAndUploadLogfiles, based on this setting
		transientLogState.LogFiles[idx].FilterLogSecret = state.ParseFilterLogSecret(server.Config.FilterLogSecret)
	}

	if globalCollectionOpts.DebugLogs {
		logger.PrintInfo("Would have sent log state:\n")
		for _, logFile := range transientLogState.LogFiles {
			content, _ := ioutil.ReadFile(logFile.TmpFile.Name())
			logs.PrintDebugInfo(string(content), logFile.LogLines, transientLogState.QuerySamples)
		}
		return nil
	}

	err = output.UploadAndSendLogs(ctx, server, grant, globalCollectionOpts, logger, transientLogState)
	if err != nil {
		return errors.Wrap(err, "failed to upload/send logs")
	}

	return nil
}

// TestLogsForAllServers - Test log download/tailing
func TestLogsForAllServers(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (hasFailedServers bool, hasSuccessfulLocalServers bool) {
	if !globalCollectionOpts.TestRun {
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

		logLinePrefix, err := postgres.GetPostgresSetting(ctx, "log_line_prefix", server, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("ERROR - Could not check log_line_prefix for server: %s", err)
			hasFailedServers = true
			continue
		} else if server.Config.SystemType == "heroku" && logLinePrefix == logs.LogPrefixHerokuHobbyTier {
			prefixedLogger.PrintWarning("WARNING - Detected log_line_prefix indicates Heroku Postgres Hobby tier, which has no log output support")
			continue
		} else if !logs.IsSupportedPrefix(logLinePrefix) {
			prefixedLogger.PrintError("ERROR - Unsupported log_line_prefix setting: '%s'", logLinePrefix)
			prefixedLogger.PrintInfo("HINT - You can find a list of supported settings in the pganalyze documentation: https://pganalyze.com/docs/log-insights/setup/self-managed/troubleshooting")
			hasFailedServers = true
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
			if testLocalLogTail(ctx, &wg, server, globalCollectionOpts, prefixedLogger) {
				hasSuccessfulLocalServers = true
				success = true
			} else {
				success = false
			}
		} else if server.Config.LogKubernetesPod != "" && server.Config.LogKubernetesContainer != "" {
			success = testKubernetesLogTail(ctx, &wg, server, globalCollectionOpts, prefixedLogger)
		} else if server.Config.SupportsLogDownload() {
			success = testLogDownload(ctx, &wg, server, globalCollectionOpts, prefixedLogger)
		} else if server.Config.AzureDbServerName != "" && server.Config.AzureEventhubNamespace != "" && server.Config.AzureEventhubName != "" {
			success = testAzureLogStream(ctx, &wg, server, globalCollectionOpts, prefixedLogger)
		} else if server.Config.GcpCloudSQLInstanceID != "" && server.Config.GcpPubsubSubscription != "" {
			success = testGoogleCloudsqlLogStream(ctx, &wg, server, globalCollectionOpts, prefixedLogger)
		}

		if !success {
			hasFailedServers = true
		}

		cancel()
	}

	return
}

func testLocalLogTail(ctx context.Context, wg *sync.WaitGroup, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) bool {
	logger.PrintInfo("Testing log collection (local)...")

	logTestSucceeded := make(chan bool, 1)
	parsedLogStream := setupLogStreamer(ctx, wg, globalCollectionOpts, logger, []*state.Server{server}, logTestSucceeded, stream.LogTestCollectorIdentify)

	err := selfhosted.SetupLogTailForServer(ctx, wg, globalCollectionOpts, logger, server, parsedLogStream)
	if err != nil {
		logger.PrintError("ERROR - Could not tail logs for server: %s", err)
		return false
	}

	EmitTestLogMsg(ctx, server, globalCollectionOpts, logger)

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

func testKubernetesLogTail(ctx context.Context, wg *sync.WaitGroup, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) bool {
	logger.PrintInfo("Testing log collection (kubernetes)...")

	logTestSucceeded := make(chan bool, 1)
	parsedLogStream := setupLogStreamer(ctx, wg, globalCollectionOpts, logger, []*state.Server{server}, logTestSucceeded, stream.LogTestCollectorIdentify)

	err := selfhosted.SetupLogTailForPod(ctx, wg, globalCollectionOpts, logger, server, parsedLogStream)
	if err != nil {
		logger.PrintError("ERROR - Could not tail logs for server: %s", err)
		return false
	}

	logger.PrintInfo("  Kubernetes log test successful")
	return true
}

func testLogDownload(ctx context.Context, wg *sync.WaitGroup, server *state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger) bool {
	prefixedLogger.PrintInfo("Testing log download...")
	_, _, err := downloadLogsForServer(ctx, server, globalCollectionOpts, prefixedLogger)
	if err != nil {
		printLogDownloadError(server, err, prefixedLogger)
		return false
	}

	prefixedLogger.PrintInfo("  Log test successful")
	return true
}

func testAzureLogStream(ctx context.Context, wg *sync.WaitGroup, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) bool {
	logger.PrintInfo("Testing log collection (Azure Database)...")

	logTestSucceeded := make(chan bool, 1)
	parsedLogStream := setupLogStreamer(ctx, wg, globalCollectionOpts, logger, []*state.Server{server}, logTestSucceeded, stream.LogTestAnyEvent)

	err := azure.SetupLogSubscriber(ctx, wg, globalCollectionOpts, logger, []*state.Server{server}, parsedLogStream)
	if err != nil {
		logger.PrintError("ERROR - Could not get logs through Azure Event Hub: %s", err)
		if strings.HasPrefix(err.Error(), "failed to configure Azure AD JWT provider: failed") {
			logger.PrintInfo("HINT - This may occur when you have multiple user-assigned managed identities set for your virtual machine. Try removing any unrelated managed identities, or explicitly set the azure_ad_client_id setting to the managed identity's Client ID.")
		}
		return false
	}

	EmitTestLogMsg(ctx, server, globalCollectionOpts, logger)

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

func testGoogleCloudsqlLogStream(ctx context.Context, wg *sync.WaitGroup, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) bool {
	logger.PrintInfo("Testing log collection (Google Cloud SQL)...")

	logTestSucceeded := make(chan bool, 1)
	parsedLogStream := setupLogStreamer(ctx, wg, globalCollectionOpts, logger, []*state.Server{server}, logTestSucceeded, stream.LogTestCollectorIdentify)

	err := google_cloudsql.SetupLogSubscriber(ctx, wg, globalCollectionOpts, logger, []*state.Server{server}, parsedLogStream)
	if err != nil {
		logger.PrintError("ERROR - Could not get logs through Google Cloud Pub/Sub: %s", err)
		return false
	}

	EmitTestLogMsg(ctx, server, globalCollectionOpts, logger)

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

func printLogDownloadError(server *state.Server, err error, prefixedLogger *util.Logger) {
	prefixedLogger.PrintError("ERROR - Could not download logs: %s", err)
	msg := err.Error()
	if server.Config.SystemType == "amazon_rds" && strings.Contains(msg, "NoCredentialProviders") {
		prefixedLogger.PrintInfo("HINT - This may occur if you have not assigned an IAM role to the collector EC2 instance, and have not provided AWS credentials through another method")
	}
}
