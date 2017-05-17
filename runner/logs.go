package runner

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

func processLogsForServer(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (bool, error) {
	grant, err := getLogsGrant(server, globalCollectionOpts, logger)
	if err != nil {
		return false, errors.Wrap(err, "could not get log grant")
	}

	if !grant.Valid {
		logger.PrintVerbose("Log collection disabled from server, skipping")
		return false, nil
	}

	// TODO: We'll need to pass a connection here for EXPLAINs to run (or hand them over to the next full snapshot run)
	logState, err := input.CollectLogs(server, nil, globalCollectionOpts, logger)
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

func getLogsGrant(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.GrantLogs, error) {
	req, err := http.NewRequest("GET", server.Config.APIBaseURL+"/v2/snapshots/grant_logs", nil)
	if err != nil {
		return state.GrantLogs{}, err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return state.GrantLogs{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return state.GrantLogs{}, err
	}

	if resp.StatusCode == http.StatusForbidden {
		return state.GrantLogs{}, nil
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return state.GrantLogs{}, fmt.Errorf("Error when getting grant: %s", body)
	}

	grant := state.GrantLogs{}
	err = json.Unmarshal(body, &grant)
	if err != nil {
		return state.GrantLogs{}, err
	}
	grant.Valid = true

	return grant, nil
}
