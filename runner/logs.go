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
)

// CollectLogsFromAllServers - Collects logs from all servers and sends them to the pganalyze service
func CollectLogsFromAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	if !globalCollectionOpts.CollectLogs {
		return
	}

	for _, server := range servers {
		var err error

		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

		grant, err := getLogsGrant(server, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Could not get log grant: %s", err)
			continue
		}

		if !grant.Valid {
			prefixedLogger.PrintVerbose("Log collection disabled, skipping")
			continue
		}

		// TODO: We'll need to pass a connection here for EXPLAINs to run (or hand them over to the next full snapshot run)
		logState, err := input.CollectLogs(server, nil, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Could not collect logs: %s", err)
			continue
		}

		err = output.UploadAndSendLogs(server, grant, globalCollectionOpts, prefixedLogger, logState)
		if err != nil {
			prefixedLogger.PrintError("Failed to upload/send logs: %s", err)
			continue
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
