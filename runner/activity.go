package runner

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

func processActivityForServer(server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedActivityState, bool, error) {
	var newGrant state.Grant
	var err error
	var connection *sql.DB
	var activity state.TransientActivityState

	newState := server.ActivityPrevState

	if server.Config.SkipIfReplica {
		connection, err = postgres.EstablishConnection(server, logger, globalCollectionOpts, "")
		if err != nil {
			return newState, false, errors.Wrap(err, "failed to connect to database")
		}
		defer connection.Close()
		var isReplica bool
		isReplica, err = postgres.GetIsReplica(logger, connection)
		if err != nil {
			return newState, false, err
		}
		if isReplica {
			return newState, false, state.ErrReplicaCollectionDisabled
		}
	}

	if !globalCollectionOpts.ForceEmptyGrant {
		newGrant, err = grant.GetDefaultGrant(server, globalCollectionOpts, logger)
		if err != nil {
			return newState, false, errors.Wrap(err, "could not get default grant for activity snapshot")
		}

		if !newGrant.Config.EnableActivity {
			if globalCollectionOpts.TestRun {
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
		connection, err = postgres.EstablishConnection(server, logger, globalCollectionOpts, "")
		if err != nil {
			return newState, false, errors.Wrap(err, "failed to connect to database")
		}
		defer connection.Close()
	}

	trackActivityQuerySize, err := postgres.GetPostgresSetting("track_activity_query_size", server, globalCollectionOpts, logger)
	if err != nil {
		activity.TrackActivityQuerySize = -1
	} else {
		activity.TrackActivityQuerySize, err = strconv.Atoi(trackActivityQuerySize)
		if err != nil {
			activity.TrackActivityQuerySize = -1
		}
	}

	activity.Version, err = postgres.GetPostgresVersion(logger, connection)
	if err != nil {
		return newState, false, errors.Wrap(err, "error collecting postgres version")
	}

	if activity.Version.Numeric < state.MinRequiredPostgresVersion {
		return newState, false, fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required", activity.Version.Short)
	}

	activity.Backends, err = postgres.GetBackends(logger, connection, activity.Version, server.Config.SystemType)
	if err != nil {
		return newState, false, errors.Wrap(err, "error collecting pg_stat_activity")
	}

	activity.Vacuums, err = postgres.GetVacuumProgress(logger, connection, activity.Version, server.Config.IgnoreSchemaRegexp)
	if err != nil {
		return newState, false, errors.Wrap(err, "error collecting pg_stat_vacuum_progress")
	}

	activity.CollectedAt = time.Now()

	err = output.SubmitCompactActivitySnapshot(server, newGrant, globalCollectionOpts, logger, activity)
	if err != nil {
		return newState, false, errors.Wrap(err, "failed to upload/send activity snapshot")
	}
	newState.ActivitySnapshotAt = activity.CollectedAt

	return newState, true, nil
}

// CollectActivityFromAllServers - Collects activity from all servers and sends them to the pganalyze service
func CollectActivityFromAllServers(servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (allSuccessful bool) {
	var wg sync.WaitGroup

	allSuccessful = true

	for idx := range servers {
		if servers[idx].Config.DisableActivity || (servers[idx].Grant.Valid && !servers[idx].Grant.Config.EnableActivity) {
			continue
		}

		wg.Add(1)
		go func(server *state.Server) {
			prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

			if globalCollectionOpts.TestRun {
				prefixedLogger.PrintInfo("Testing activity snapshots...")
			}

			server.ActivityStateMutex.Lock()
			newState, success, err := processActivityForServer(server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				server.ActivityStateMutex.Unlock()

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
				server.ActivityPrevState = newState
				server.ActivityStateMutex.Unlock()
				if success && server.Config.SuccessCallback != "" {
					go runCompletionCallback("success", server.Config.SuccessCallback, server.Config.SectionName, "activity", nil, prefixedLogger)
				}
			}
			wg.Done()
		}(servers[idx])
	}

	wg.Wait()

	return
}
