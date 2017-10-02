package runner

import (
	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

func processLogsForServer(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (bool, error) {
	grant, err := grant.GetLogsGrant(server, globalCollectionOpts, logger)
	if err != nil {
		return false, errors.Wrap(err, "could not get log grant")
	}

	if !grant.Valid {
		logger.PrintVerbose("Log collection disabled from server, skipping")
		return false, nil
	}

	// TODO: We'll need to pass a connection here for EXPLAINs to run (or hand them over to the next full snapshot run)
	logState, err := input.CollectLogs(server, nil, globalCollectionOpts, logger)
	defer logState.Cleanup()
	if err != nil {
		return false, errors.Wrap(err, "could not collect logs")
	}

	err = output.UploadAndSendLogs(server, grant, globalCollectionOpts, logger, logState)
	if err != nil {
		return false, errors.Wrap(err, "failed to upload/send logs")
	}

	return true, nil
}

// CollectLogsFromAllServers - Collects logs from all servers and sends them to the pganalyze service
func CollectLogsFromAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	if !globalCollectionOpts.CollectLogs {
		return
	}

	for _, server := range servers {
		if !server.Config.EnableLogs {
			continue
		}

		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

		success, err := processLogsForServer(server, globalCollectionOpts, prefixedLogger)
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
