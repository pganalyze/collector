package runner

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func processDatabase(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedState, state.Grant, error) {
	var grant state.Grant
	var err error

	if globalCollectionOpts.SubmitCollectedData {
		// Note: In case of server errors, we should reuse the old grant if its still recent (i.e. less than 50 minutes ago)
		grant, err = getSnapshotGrant(server, globalCollectionOpts, logger)
		if err != nil {
			if server.Grant.Valid {
				logger.PrintVerbose("Could not acquire snapshot grant, reusing previous grant: %s", err)
			} else {
				return state.PersistedState{}, state.Grant{}, err
			}
		} else {
			server.Grant = grant
		}
	}

	newState, transientState, err := input.CollectFull(server, globalCollectionOpts, logger)
	if err != nil {
		return newState, grant, err
	}

	collectedIntervalSecs := uint32(600) // TODO: 10 minutes - we should actually measure the distance between states here
	diffState := diffState(logger, server.PrevState, newState, collectedIntervalSecs)

	output.SendFull(server, globalCollectionOpts, logger, newState, diffState, transientState, collectedIntervalSecs)

	return newState, grant, nil
}

func getSnapshotGrant(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.Grant, error) {
	req, err := http.NewRequest("GET", server.Config.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		return state.Grant{}, err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Add("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return state.Grant{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return state.Grant{}, err
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		return state.Grant{}, fmt.Errorf("Error when getting grant: %s\n", body)
	}

	grant := state.Grant{}
	err = json.Unmarshal(body, &grant)
	if err != nil {
		return state.Grant{}, err
	}
	grant.Valid = true

	return grant, nil
}

func writeStateFile(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	stateOnDisk := state.StateOnDisk{PrevStateByAPIKey: make(map[string]state.PersistedState), FormatVersion: state.StateOnDiskFormatVersion}

	for _, server := range servers {
		stateOnDisk.PrevStateByAPIKey[server.Config.APIKey] = server.PrevState
	}

	file, err := os.Create(globalCollectionOpts.StateFilename)
	if err != nil {
		logger.PrintWarning("Could not write out state file to %s because of error: %s", globalCollectionOpts.StateFilename, err)
		return
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	encoder.Encode(stateOnDisk)
}

// ReadStateFile - This reads in the prevState structs from the state file - only run this on initial bootup and SIGHUP!
func ReadStateFile(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	var stateOnDisk state.StateOnDisk

	file, err := os.Open(globalCollectionOpts.StateFilename)
	if err != nil {
		logger.PrintVerbose("Did not open state file: %s", err)
		return
	}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&stateOnDisk)
	if err != nil {
		logger.PrintVerbose("Could not decode state file: %s", err)
		return
	}
	defer file.Close()

	if stateOnDisk.FormatVersion < state.StateOnDiskFormatVersion {
		logger.PrintVerbose("Ignoring state file since the on-disk format has changed")
		return
	}

	for idx, server := range servers {
		prevState, exist := stateOnDisk.PrevStateByAPIKey[server.Config.APIKey]
		if exist {
			prefixedLogger := logger.WithPrefix(server.Config.SectionName)
			prefixedLogger.PrintVerbose("Successfully recovered state from on-disk file")
			servers[idx].PrevState = prevState
		}
	}
}

// CollectAllServers - Collects statistics from all servers and sends them as full snapshots to the pganalyze service
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

	if globalCollectionOpts.WriteStateUpdate {
		writeStateFile(servers, globalCollectionOpts, logger)
	}
}
