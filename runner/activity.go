package runner

import (
	"context"
	"database/sql"
	"strconv"
	"sync"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

func processActivityForServer(ctx context.Context, server *state.Server, opts state.CollectionOpts, logger *util.Logger) (state.PersistedActivityState, bool, error) {
	var newGrant state.Grant
	var err error
	var connection *sql.DB
	var activity state.TransientActivityState

	newState := server.ActivityPrevState

	if server.Pause.Load() {
		logger.PrintVerbose("Duplicate collector detected: Please ensure only one collector is monitoring this Postgres server")
		return newState, false, nil
	}

	if server.Config.SkipIfReplica {
		connection, err = postgres.EstablishConnection(ctx, server, logger, opts, "")
		if err != nil {
			return newState, false, errors.Wrap(err, "failed to connect to database")
		}
		defer connection.Close()
		var isReplica bool
		isReplica, err = postgres.GetIsReplica(ctx, logger, connection)
		if err != nil {
			return newState, false, err
		}
		if isReplica {
			return newState, false, state.ErrReplicaCollectionDisabled
		}
	}

	if server.WebSocket.Load() != nil {
		newGrant = *server.Grant.Load()
	}

	if !newGrant.Valid && !opts.ForceEmptyGrant {
		newGrant, err = output.GetGrant(ctx, server, opts, logger)
		if err != nil {
			return newState, false, errors.Wrap(err, "could not get default grant for activity snapshot")
		}

		if !newGrant.Config.EnableActivity {
			if opts.TestRun {
				server.SelfTest.MarkCollectionAspectNotAvailable(state.CollectionAspectActivity, "not available on this plan")
				server.SelfTest.HintCollectionAspect(state.CollectionAspectActivity, "Compare plans at %s", selftest.URLPrinter.Sprint("https://pganalyze.com/pricing"))
				logger.PrintError("  Failed - Activity snapshots disabled by pganalyze")
			} else {
				logger.PrintVerbose("Activity snapshots disabled by pganalyze, skipping")
			}
			return newState, false, nil
		}
	}
	// N.B.: Without the SkipIfReplica flag, we wait to establish the connection to avoid opening
	// and closing it for no reason when the grant EnableActivity is not set (e.g., production plans)
	if connection == nil {
		connection, err = postgres.EstablishConnection(ctx, server, logger, opts, "")
		if err != nil {
			return newState, false, errors.Wrap(err, "failed to connect to database")
		}
		defer connection.Close()
	}

	trackActivityQuerySize, err := postgres.GetPostgresSetting(ctx, connection, "track_activity_query_size")
	if err != nil {
		activity.TrackActivityQuerySize = -1
	} else {
		activity.TrackActivityQuerySize, err = strconv.Atoi(trackActivityQuerySize)
		if err != nil {
			activity.TrackActivityQuerySize = -1
		}
	}

	c, err := postgres.NewCollection(ctx, logger, server, opts, connection)
	if err != nil {
		return newState, false, err
	}

	activity.Backends, err = postgres.GetBackends(ctx, c, connection)
	if err != nil {
		return newState, false, errors.Wrap(err, "error collecting pg_stat_activity")
	}

	activity.Vacuums, err = postgres.GetVacuumProgress(ctx, c, connection)
	if err != nil {
		return newState, false, errors.Wrap(err, "error collecting pg_stat_vacuum_progress")
	}

	activity.CollectedAt = time.Now()

	err = output.SubmitCompactActivitySnapshot(ctx, server, newGrant, opts, logger, activity)
	if err != nil {
		return newState, false, errors.Wrap(err, "failed to upload/send activity snapshot")
	}
	newState.ActivitySnapshotAt = activity.CollectedAt

	return newState, true, nil
}

// CollectActivityFromAllServers - Collects activity from all servers and sends them to the pganalyze service
func CollectActivityFromAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) (allSuccessful bool) {
	var wg sync.WaitGroup

	allSuccessful = true

	for idx := range servers {
		server := servers[idx]
		grant := server.Grant.Load()
		if server.Config.DisableActivity || (grant.Valid && !grant.Config.EnableActivity) {
			server.SelfTest.MarkCollectionAspectNotAvailable(state.CollectionAspectActivity, "not available on this plan")
			server.SelfTest.HintCollectionAspect(state.CollectionAspectActivity, "Compare plans at %s", selftest.URLPrinter.Sprint("https://pganalyze.com/pricing"))
			continue
		}

		wg.Add(1)
		go func(server *state.Server) {
			prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

			if opts.TestRun {
				prefixedLogger.PrintInfo("Testing activity snapshots...")
			}

			server.ActivityStateMutex.Lock()
			newState, success, err := processActivityForServer(ctx, server, opts, prefixedLogger)
			if err != nil {
				server.ActivityStateMutex.Unlock()
				server.SelfTest.MarkCollectionAspectError(state.CollectionAspectActivity, err.Error())

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
					prefixedLogger.PrintError("Could not collect activity for server: %s", err)
					if !isIgnoredReplica && server.Config.ErrorCallback != "" {
						go runCompletionCallback("error", server.Config.ErrorCallback, server.Config.SectionName, "activity", err, prefixedLogger)
					}
				}
			} else {
				if success {
					server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectActivity)
				}
				server.ActivityPrevState = newState
				server.ActivityStateMutex.Unlock()
				if success && server.Config.SuccessCallback != "" {
					go runCompletionCallback("success", server.Config.SuccessCallback, server.Config.SectionName, "activity", nil, prefixedLogger)
				}
			}
			wg.Done()
		}(server)
	}

	wg.Wait()

	return
}
