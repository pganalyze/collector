package runner

import (
	"sync"

	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/input/system/google_cloudsql"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

func downloadLogsForServer(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (bool, error) {
	grant, err := grant.GetLogsGrant(server, globalCollectionOpts, logger)
	if err != nil {
		return false, errors.Wrap(err, "could not get log grant")
	}

	if !grant.Valid {
		if globalCollectionOpts.TestRun {
			logger.PrintError("  Failed - Log collection disabled by pganalyze")
		} else {
			logger.PrintVerbose("Log collection disabled by pganalyze, skipping")
		}
		return false, nil
	}

	// TODO: We'll need to pass a connection here for EXPLAINs to run (or hand them over to the next full snapshot run)
	logState, err := input.DownloadLogs(server, nil, globalCollectionOpts, logger)
	if err != nil {
		logState.Cleanup()
		return false, errors.Wrap(err, "could not collect logs")
	}

	err = output.UploadAndSendLogs(server, grant, globalCollectionOpts, logger, logState)
	if err != nil {
		logState.Cleanup()
		return false, errors.Wrap(err, "failed to upload/send logs")
	}

	logState.Cleanup()
	return true, nil
}

// TestLogsForAllServers - Test log download/tailing
func TestLogsForAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (hasFailedServers bool, hasSuccessfulLocalServers bool) {
	if !globalCollectionOpts.TestRun {
		return
	}

	for _, server := range servers {
		if server.Config.DisableLogs {
			continue
		}

		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

		if server.Config.LogLocation != "" {
			prefixedLogger.PrintInfo("Testing log collection (local)...")
			err := selfhosted.TestLogTail(server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				hasFailedServers = true
				prefixedLogger.PrintError("ERROR - Could not tail logs for server: %s", err)
			} else {
				prefixedLogger.PrintInfo("  Local log test successful")
				hasSuccessfulLocalServers = true
			}
		} else if server.Config.GcpCloudSQLInstanceID != "" && server.Config.GcpPubsubSubscription != "" {
			prefixedLogger.PrintInfo("Testing log collection (Google Cloud SQL)...")
			err := google_cloudsql.LogTestRun(server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				hasFailedServers = true
				prefixedLogger.PrintError("ERROR - Could not get Pub/Sub log output for server: %s", err)
			} else {
				prefixedLogger.PrintInfo("  Log test successful")
			}
		} else if server.Config.AwsDbInstanceID != "" {
			prefixedLogger.PrintInfo("Testing log collection (RDS)...")
			_, err := downloadLogsForServer(server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				hasFailedServers = true
				prefixedLogger.PrintError("Could not download logs for server: %s", err)
			}
		}
	}

	return
}

// DownloadLogsFromAllServers - Downloads logs from all servers that are remote systems and sends them to the pganalyze service
func DownloadLogsFromAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	var wg sync.WaitGroup

	if !globalCollectionOpts.CollectLogs {
		return
	}

	for idx := range servers {
		if servers[idx].Config.DisableLogs || (servers[idx].Grant.Valid && !servers[idx].Grant.Config.EnableLogs) {
			continue
		}

		if servers[idx].Config.AwsDbInstanceID == "" {
			continue
		}

		wg.Add(1)
		go func(server *state.Server) {
			prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

			success, err := downloadLogsForServer(*server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("Could not collect logs for server: %s", err)
				if server.Config.ErrorCallback != "" {
					go runCompletionCallback("error", server.Config.ErrorCallback, server.Config.SectionName, "logs", err, prefixedLogger)
				}
			} else if success {
				if server.Config.SuccessCallback != "" {
					go runCompletionCallback("success", server.Config.SuccessCallback, server.Config.SectionName, "logs", nil, prefixedLogger)
				}
			}
			wg.Done()
		}(&servers[idx])
	}

	wg.Wait()

	return
}
