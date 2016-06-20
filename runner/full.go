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

func processDatabase(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.State, state.Grant, error) {
	// Note: In case of server errors, we should reuse the old grant if its still recent (i.e. less than 50 minutes ago)
	grant, err := getSnapshotGrant(server, globalCollectionOpts, logger)
	if err != nil {
		logger.PrintVerbose("Could not acquire snapshot grant, reusing previous grant: %s", err)
	} else {
		server.Grant = grant
	}

	newState, err := input.CollectFull(server, globalCollectionOpts, logger)
	if err != nil {
		return newState, grant, err
	}

	diffState := diffState(logger, server.PrevState, newState)

	output.SendFull(server, globalCollectionOpts, logger, newState, diffState)

	return newState, grant, nil
}

func getSnapshotGrant(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.Grant, error) {
	var grant state.Grant

	req, err := http.NewRequest("GET", server.Config.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		return grant, err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return grant, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return grant, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return grant, fmt.Errorf("Error when getting grant: %s\n", body)
	}

	err = json.Unmarshal(body, &grant)
	if err != nil {
		return grant, err
	}

	return grant, nil
}

func CollectAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for idx, server := range servers {
		var err error

		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		server.Connection, err = establishConnection(server, logger, globalCollectionOpts)
		if err != nil {
			prefixedLogger.PrintError("Error: Failed to connect to database: %s", err)
			return
		}

		newState, grant, err := processDatabase(server, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Error: Could not process database: %s", err)
		} else {
			servers[idx].Grant = grant
			servers[idx].PrevState = newState
		}

		// This is the easiest way to avoid opening multiple connections to different databases on the same instance
		server.Connection.Close()
		server.Connection = nil
	}
}
