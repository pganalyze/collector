package runner

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

func processActivityForServer(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (bool, error) {
	var newGrant state.Grant
	var err error
	var connection *sql.DB
	var activity state.ActivityState

	if !globalCollectionOpts.ForceEmptyGrant {
		newGrant, err = grant.GetDefaultGrant(server, globalCollectionOpts, logger)
		if err != nil {
			return false, errors.Wrap(err, "could not get default grant for activity snapshot")
		}

		if !newGrant.Config.EnableActivity {
			if globalCollectionOpts.TestRun {
				logger.PrintError("  Failed - Activity snapshots disabled by pganalyze")
			} else {
				logger.PrintVerbose("Activity snapshots disabled by pganalyze, skipping")
			}
			return false, nil
		}
	}

	connection, err = postgres.EstablishConnection(server, logger, globalCollectionOpts, "")
	if err != nil {
		return false, errors.Wrap(err, "failed to connect to database")
	}

	defer connection.Close()

	activity.Version, err = postgres.GetPostgresVersion(logger, connection)
	if err != nil {
		return false, errors.Wrap(err, "error collecting postgres version")
	}

	if activity.Version.Numeric < state.MinRequiredPostgresVersion {
		return false, fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required", activity.Version.Short)
	}

	activity.Backends, err = postgres.GetBackends(logger, connection, activity.Version)
	if err != nil {
		return false, errors.Wrap(err, "error collecting pg_stat_activity")
	}

	activity.Vacuums, err = postgres.GetVacuumProgress(logger, connection, activity.Version)
	if err != nil {
		return false, errors.Wrap(err, "error collecting pg_stat_vacuum_progress")
	}

	activity.CollectedAt = time.Now()

	err = output.SubmitCompactActivitySnapshot(server, newGrant, globalCollectionOpts, logger, activity)
	if err != nil {
		return false, errors.Wrap(err, "failed to upload/send activity snapshot")
	}

	return true, nil
}

// CollectActivityFromAllServers - Collects activity from all servers and sends them to the pganalyze service
func CollectActivityFromAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	var wg sync.WaitGroup

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

			success, err := processActivityForServer(*server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("Could not collect activity for server: %s", err)
				if server.Config.ErrorCallback != "" {
					go runCompletionCallback("error", server.Config.ErrorCallback, server.Config.SectionName, "activity", err, prefixedLogger)
				}
			} else if success {
				if server.Config.SuccessCallback != "" {
					go runCompletionCallback("success", server.Config.SuccessCallback, server.Config.SectionName, "activity", nil, prefixedLogger)
				}
			}
			wg.Done()
		}(&servers[idx])
	}

	wg.Wait()

	return
}
