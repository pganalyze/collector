package runner

import (
	"database/sql"
	"fmt"
	"os/exec"
	"runtime/debug"
	"sync"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func collectDiffAndSubmit(server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedState, state.CollectionStatus, error) {
	var newState state.PersistedState
	var err error
	var connection *sql.DB

	connection, err = postgres.EstablishConnection(server, logger, globalCollectionOpts, "")
	if err != nil {
		return newState, state.CollectionStatus{}, fmt.Errorf("Failed to connect to database: %s", err)
	}

	newState, transientState, err := input.CollectFull(server, connection, globalCollectionOpts, logger)
	if err != nil {
		connection.Close()
		return newState, state.CollectionStatus{}, err
	}

	// This is the easiest way to avoid opening multiple connections to different databases on the same instance
	connection.Close()

	logsDisabled, logsDisabledReason := logs.ValidateLogCollectionConfig(server, transientState.Settings)
	collectionStatus := state.CollectionStatus{
		LogSnapshotDisabled:       logsDisabled,
		LogSnapshotDisabledReason: logsDisabledReason,
	}

	collectedIntervalSecs := uint32(newState.CollectedAt.Sub(server.PrevState.CollectedAt) / time.Second)
	if collectedIntervalSecs == 0 {
		collectedIntervalSecs = 1 // Avoid divide by zero errors for fast consecutive runs
	}

	diffState := diffState(logger, server.PrevState, newState, collectedIntervalSecs)

	transientState.HistoricStatementStats = server.PrevState.UnidentifiedStatementStats

	err = output.SendFull(server, globalCollectionOpts, logger, newState, diffState, transientState, collectedIntervalSecs)
	if err != nil {
		return newState, collectionStatus, err
	}

	// After we've done all processing, and in case we did a reset, make sure the
	// next snapshot has an empty reference point
	if transientState.ResetStatementStats != nil {
		newState.StatementStats = transientState.ResetStatementStats
	}

	return newState, collectionStatus, nil
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

func processServer(server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedState, state.Grant, state.CollectionStatus, error) {
	var newGrant state.Grant
	var newState state.PersistedState
	var collectionStatus state.CollectionStatus
	var err error

	if !globalCollectionOpts.ForceEmptyGrant {
		// Note: In case of server errors, we should reuse the old grant if its still recent (i.e. less than 50 minutes ago)
		newGrant, err = grant.GetDefaultGrant(server, globalCollectionOpts, logger)
		if err != nil {
			if server.Grant.Valid {
				logger.PrintVerbose("Could not acquire snapshot grant, reusing previous grant: %s", err)
			} else {
				return state.PersistedState{}, state.Grant{}, state.CollectionStatus{}, err
			}
		} else {
			server.Grant = newGrant
		}
	}

	var sentryClient *raven.Client
	if server.Grant.Config.SentryDsn != "" {
		sentryClient, err = raven.NewWithTags(server.Grant.Config.SentryDsn, map[string]string{"server_id": server.Grant.Config.ServerID})
		if err != nil {
			logger.PrintVerbose("Failed to setup Sentry client: %s", err)
		} else {
			sentryClient.SetRelease(util.CollectorVersion)
		}
	}

	runFunc := func() {
		newState, collectionStatus, err = collectDiffAndSubmit(server, globalCollectionOpts, logger)
	}

	var panicErr interface{}
	var stackTrace []byte
	if sentryClient != nil {
		panicErr, _ = sentryClient.CapturePanic(runFunc, nil)
		sentryClient.Wait()
	} else {
		panicErr, stackTrace = capturePanic(runFunc)
	}
	if panicErr != nil {
		err = fmt.Errorf("%s", panicErr)
		logger.PrintVerbose("Panic: %s\n%s", err, stackTrace)
	}

	return newState, newGrant, collectionStatus, err
}

func runCompletionCallback(callbackType string, callbackCmd string, sectionName string, snapshotType string, errIn error, logger *util.Logger) {
	cmd := exec.Command("bash", "-c", callbackCmd)
	cmd.Env = append(cmd.Env, "PGA_CALLBACK_TYPE="+callbackType)
	cmd.Env = append(cmd.Env, "PGA_CONFIG_SECTION="+sectionName)
	cmd.Env = append(cmd.Env, "PGA_SNAPSHOT_TYPE="+snapshotType)
	if errIn != nil {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGA_ERROR_MESSAGE=%s", errIn))
	}
	err := cmd.Run()
	if err != nil {
		logger.PrintError("Could not run %s callback (%s snapshot): %s", callbackType, snapshotType, callbackCmd)
	}
}

// CollectAllServers - Collects statistics from all servers and sends them as full snapshots to the pganalyze service
func CollectAllServers(servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (allSuccessful bool) {
	var wg sync.WaitGroup

	allSuccessful = true

	for idx := range servers {
		wg.Add(1)
		go func(server *state.Server) {
			var err error

			prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

			if globalCollectionOpts.TestRun {
				prefixedLogger.PrintInfo("Testing statistics collection...")
			}

			server.StateMutex.Lock()
			newState, grant, newCollectionStatus, err := processServer(server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				server.StateMutex.Unlock()
				allSuccessful = false
				prefixedLogger.PrintError("Could not process server: %s", err)
				if grant.Valid && !globalCollectionOpts.TestRun && globalCollectionOpts.SubmitCollectedData {
					server.Grant = grant
					err = output.SendFailedFull(server, globalCollectionOpts, prefixedLogger)
					if err != nil {
						prefixedLogger.PrintWarning("Could not send error information to remote server: %s", err)
					}
				}
				if server.Config.ErrorCallback != "" {
					go runCompletionCallback("error", server.Config.ErrorCallback, server.Config.SectionName, "full", err, prefixedLogger)
				}
			} else {
				server.Grant = grant
				server.PrevState = newState
				server.StateMutex.Unlock()
				server.CollectionStatusMutex.Lock()
				if newCollectionStatus.LogSnapshotDisabled {
					warning := fmt.Sprintf("Skipping logs: %s", server.CollectionStatus.LogSnapshotDisabledReason)
					prefixedLogger.PrintWarning(warning)
				}
				server.CollectionStatus = newCollectionStatus
				server.CollectionStatusMutex.Unlock()
				if server.Config.SuccessCallback != "" {
					go runCompletionCallback("success", server.Config.SuccessCallback, server.Config.SectionName, "full", nil, prefixedLogger)
				}
			}
			wg.Done()
		}(servers[idx])
	}

	wg.Wait()

	if globalCollectionOpts.WriteStateUpdate {
		state.WriteStateFile(servers, globalCollectionOpts, logger)
	}

	return
}
