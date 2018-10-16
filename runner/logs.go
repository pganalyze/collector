package runner

import (
	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input"
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
		logger.PrintVerbose("Log collection disabled from server, skipping")
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
func TestLogsForAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (hasSuccessfulLocalServers bool) {
	if !globalCollectionOpts.TestRun {
		return
	}

	for _, server := range servers {
		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

		if server.Config.LogLocation != "" {
			prefixedLogger.PrintInfo("Testing local log tailing...")
			err := selfhosted.TestLogTail(server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - Could not tail logs for server: %s", err)
			} else {
				prefixedLogger.PrintInfo("Log test successful")
				hasSuccessfulLocalServers = true
			}
		} else if server.Config.EnableLogs {
			prefixedLogger.PrintInfo("Testing log download...")
			_, err := downloadLogsForServer(server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("Could not download logs for server: %s", err)
			} else {
				prefixedLogger.PrintInfo("Log test successful")
			}
		}
	}

	return
}

// DownloadLogsFromAllServers - Downloads logs from all servers that are remote systems and sends them to the pganalyze service
func DownloadLogsFromAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	if !globalCollectionOpts.CollectLogs {
		return
	}

	for _, server := range servers {
		if !server.Config.EnableLogs {
			continue
		}

		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

		success, err := downloadLogsForServer(server, globalCollectionOpts, prefixedLogger)
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
	}

	return
}
