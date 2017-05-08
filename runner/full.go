package runner

import (
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func collectDiffAndSubmit(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedState, error) {
	var newState state.PersistedState
	var err error
	var connection *sql.DB

	connection, err = postgres.EstablishConnection(server, logger, globalCollectionOpts, "")
	if err != nil {
		return newState, fmt.Errorf("Failed to connect to database: %s", err)
	}

	newState, transientState, err := input.CollectFull(server, connection, globalCollectionOpts, logger)
	if err != nil {
		connection.Close()
		return newState, err
	}

	// This is the easiest way to avoid opening multiple connections to different databases on the same instance
	connection.Close()

	collectedIntervalSecs := uint32(newState.CollectedAt.Sub(server.PrevState.CollectedAt) / time.Second)
	if collectedIntervalSecs == 0 {
		collectedIntervalSecs = 1 // Avoid divide by zero errors for fast consecutive runs
	}

	diffState := diffState(logger, server.PrevState, newState, collectedIntervalSecs)

	if transientState.HasStatementText {
		transientState.HistoricStatementStats = server.PrevState.UnidentifiedStatementStats
	} else {
		timeKey := state.PostgresStatementStatsTimeKey{CollectedAt: newState.CollectedAt, CollectedIntervalSecs: collectedIntervalSecs}
		newState.UnidentifiedStatementStats = server.PrevState.UnidentifiedStatementStats
		if newState.UnidentifiedStatementStats == nil {
			newState.UnidentifiedStatementStats = make(state.HistoricStatementStatsMap)
		}
		newState.UnidentifiedStatementStats[timeKey] = diffState.StatementStats
		diffState.StatementStats = make(state.DiffedPostgresStatementStatsMap)
	}

	err = output.SendFull(server, globalCollectionOpts, logger, newState, diffState, transientState, collectedIntervalSecs)
	if err != nil {
		return newState, err
	}

	return newState, nil
}

func capturePanic(f func()) (err interface{}, stackTrace []byte) {
	defer func() {
		if err = recover(); err != nil {
			stackTrace = debug.Stack()
		}
	}()

	f()

	return
}

func processDatabase(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedState, state.Grant, error) {
	var grant state.Grant
	var newState state.PersistedState
	var err error

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

	transientState := state.TransientState{}
	if server.Grant.Config.SentryDsn != "" {
		transientState.SentryClient, err = raven.NewWithTags(server.Grant.Config.SentryDsn, map[string]string{"server_id": server.Grant.Config.ServerID})
		transientState.SentryClient.SetRelease(util.CollectorVersion)
		if err != nil {
			transientState.SentryClient = nil
			logger.PrintVerbose("Failed to setup Sentry client: %s", err)
		}
	}

	runFunc := func() {
		newState, err = collectDiffAndSubmit(server, globalCollectionOpts, logger)
	}

	var panicErr interface{}
	var stackTrace []byte
	if transientState.SentryClient != nil {
		panicErr, _ = transientState.SentryClient.CapturePanic(runFunc, nil)
		transientState.SentryClient.Wait()
		transientState.SentryClient = nil
	} else {
		panicErr, stackTrace = capturePanic(runFunc)
	}
	if panicErr != nil {
		err = fmt.Errorf("%s", panicErr)
		logger.PrintVerbose("Panic: %s\n%s", err, stackTrace)
	}

	return newState, grant, err
}

func getSnapshotGrant(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.Grant, error) {
	req, err := http.NewRequest("GET", server.Config.APIBaseURL+"/v2/snapshots/grant", nil)
	if err != nil {
		return state.Grant{}, err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
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
		return state.Grant{}, fmt.Errorf("Error when getting grant: %s", body)
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

		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

		newState, grant, err := processDatabase(server, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Could not process database: %s", err)
			if grant.Valid && !globalCollectionOpts.TestRun && globalCollectionOpts.SubmitCollectedData {
				server.Grant = grant
				err = output.SendFailedFull(server, globalCollectionOpts, prefixedLogger)
				if err != nil {
					prefixedLogger.PrintWarning("Could not send error information to remote server: %s", err)
				}
			}
		} else {
			servers[idx].Grant = grant
			servers[idx].PrevState = newState
		}
	}

	if globalCollectionOpts.WriteStateUpdate {
		writeStateFile(servers, globalCollectionOpts, logger)
	}
}
