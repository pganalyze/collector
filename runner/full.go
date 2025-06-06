package runner

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"runtime/debug"
	"sync"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/pganalyze/collector/input"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func collectDiffAndSubmit(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (state.PersistedState, state.CollectionStatus, error) {
	var newState state.PersistedState
	var err error
	var connection *sql.DB

	connection, err = postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return newState, state.CollectionStatus{}, fmt.Errorf("Failed to connect to database: %s", err)
	}

	newState, transientState, err := input.CollectFull(ctx, server, connection, opts, logger)
	if err != nil {
		connection.Close()
		return newState, state.CollectionStatus{}, err
	}
	if opts.TestRun {
		logger.PrintInfo("  Test collection successful for %s", transientState.Version.Full)
	}

	// This is the easiest way to avoid opening multiple connections to different databases on the same instance
	connection.Close()

	logsDisabled, logsIgnoreStatement, logsIgnoreDuration, logsDisabledReason := logs.ValidateLogCollectionConfig(server, transientState.Settings)
	collectionStatus := state.CollectionStatus{
		LogSnapshotDisabled:       logsDisabled,
		LogSnapshotDisabledReason: logsDisabledReason,
	}
	server.SetLogIgnoreFlags(logsIgnoreStatement, logsIgnoreDuration)

	collectedIntervalSecs := uint32(newState.CollectedAt.Sub(server.PrevState.CollectedAt) / time.Second)
	if collectedIntervalSecs == 0 {
		collectedIntervalSecs = 1 // Avoid divide by zero errors for fast consecutive runs
	}

	diffState := diffState(logger, server.PrevState, newState, collectedIntervalSecs)

	transientState.HistoricStatementStats = server.PrevState.UnidentifiedStatementStats
	transientState.HistoricPlanStats = server.PrevState.UnidentifiedPlanStats
	transientState.HistoricServerIoStats = server.PrevState.QueuedServerIoStats

	// Add current collection to historic values for easier handling (except for statements for historic reasons)
	timeKey := state.HistoricStatsTimeKey{
		CollectedAt:           newState.CollectedAt,
		CollectedIntervalSecs: uint32(newState.LastStatementStatsAt.Sub(server.PrevState.LastStatementStatsAt) / time.Second),
	}
	if transientState.HistoricServerIoStats == nil {
		transientState.HistoricServerIoStats = make(state.HistoricPostgresServerIoStatsMap)
	}
	if diffState.ServerIoStats != nil {
		transientState.HistoricServerIoStats[timeKey] = diffState.ServerIoStats // add current for easier tracking
	}

	err = output.SendFull(ctx, server, opts, logger, newState, diffState, transientState, collectedIntervalSecs)
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

func processServer(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (state.PersistedState, state.Grant, state.CollectionStatus, error) {
	var newGrant state.Grant
	var newState state.PersistedState
	var collectionStatus state.CollectionStatus
	var err error
	newGrant.Config = pganalyze_collector.ServerMessage_Config{Features: &pganalyze_collector.ServerMessage_Features{}}

	if server.Pause.Load() {
		logger.PrintWarning("Duplicate collector detected: Please ensure only one collector is monitoring this Postgres server")
		return newState, newGrant, collectionStatus, nil
	}

	err = checkReplicaCollectionDisabled(ctx, server, opts, logger)
	if err != nil {
		return newState, newGrant, collectionStatus, err
	}

	if server.WebSocket.Load() != nil {
		newGrant = *server.Grant.Load()
	} else if !opts.ForceEmptyGrant {
		newGrant, err = output.GetGrant(ctx, server, opts, logger)
		if err != nil {
			if server.Grant.Load().Valid {
				logger.PrintVerbose("Could not acquire snapshot grant, reusing previous grant: %s", err)
			} else {
				return newState, newGrant, collectionStatus, err
			}
		} else {
			server.Grant.Store(&newGrant)
		}
	}

	var sentryClient *raven.Client
	config := server.Grant.Load().Config
	if config.SentryDsn != "" {
		sentryClient, err = raven.NewWithTags(config.SentryDsn, map[string]string{"server_id": config.ServerId})
		if err != nil {
			logger.PrintVerbose("Failed to setup Sentry client: %s", err)
		} else {
			sentryClient.SetRelease(util.CollectorVersion)
		}
	}

	runFunc := func() {
		newState, collectionStatus, err = collectDiffAndSubmit(ctx, server, opts, logger)
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
		logger.PrintWarning("Panic: %s\n%s", err, stackTrace)
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

func checkReplicaCollectionDisabled(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) error {
	if !server.Config.SkipIfReplica {
		return nil
	}

	connection, err := postgres.EstablishConnection(ctx, server, logger, opts, "")
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %s", err)
	}
	defer connection.Close()

	var isReplica bool
	isReplica, err = postgres.GetIsReplica(ctx, logger, connection)
	if err != nil {
		return fmt.Errorf("Error checking replication status")
	}
	if isReplica {
		return state.ErrReplicaCollectionDisabled
	} else {
		return nil
	}
}

// CollectAllServers - Collects statistics from all servers and sends them as full snapshots to the pganalyze service
func CollectAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) (allSuccessful bool) {
	var wg sync.WaitGroup

	allSuccessful = true

	for idx := range servers {
		wg.Add(1)
		go func(server *state.Server) {
			var err error

			prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

			if opts.TestRun {
				prefixedLogger.PrintInfo("Testing statistics collection...")
			}

			server.StateMutex.Lock()
			newState, grant, newCollectionStatus, err := processServer(ctx, server, opts, prefixedLogger)
			if err != nil {
				server.StateMutex.Unlock()

				server.CollectionStatusMutex.Lock()
				isIgnoredReplica := err == state.ErrReplicaCollectionDisabled
				if isIgnoredReplica {
					reason := err.Error()
					server.CollectionStatus = state.CollectionStatus{
						CollectionDisabled:        true,
						CollectionDisabledReason:  reason,
						LogSnapshotDisabled:       true,
						LogSnapshotDisabledReason: reason,
					}
				}
				server.CollectionStatusMutex.Unlock()

				if isIgnoredReplica {
					prefixedLogger.PrintVerbose("All monitoring suspended while server is replica")
				} else {
					allSuccessful = false
					prefixedLogger.PrintError("Could not process server: %s", err)

					if grant.Valid && !opts.TestRun && opts.SubmitCollectedData {
						if server.WebSocket.Load() == nil {
							server.Grant.Store(&grant)
						}
						err = output.SendFailedFull(ctx, server, opts, prefixedLogger)
						if err != nil {
							prefixedLogger.PrintWarning("Could not send error information to remote server: %s", err)
						}
					}

					if !isIgnoredReplica && server.Config.ErrorCallback != "" {
						go runCompletionCallback("error", server.Config.ErrorCallback, server.Config.SectionName, "full", err, prefixedLogger)
					}
				}
			} else {
				if server.WebSocket.Load() == nil {
					server.Grant.Store(&grant)
				}
				server.PrevState = newState
				server.StateMutex.Unlock()
				server.CollectionStatusMutex.Lock()
				if newCollectionStatus.LogSnapshotDisabled && !opts.TestRun {
					warning := fmt.Sprintf("Skipping logs: %s", newCollectionStatus.LogSnapshotDisabledReason)
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

	if opts.WriteStateUpdate {
		state.WriteStateFile(servers, opts, logger)
	}

	return
}
