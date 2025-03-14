package input

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/scheduler"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// CollectFull - Collects a "full" snapshot of all data we need on a regular interval
func CollectFull(ctx context.Context, server *state.Server, connection *sql.DB, opts state.CollectionOpts, logger *util.Logger) (ps state.PersistedState, ts state.TransientState, err error) {
	ps.CollectedAt = time.Now()

	c, err := postgres.NewCollection(ctx, logger, server, opts, connection)
	if err != nil {
		logger.PrintError("Error setting up collection info: %s", err)
		return
	}
	ts.Version = c.PostgresVersion
	ts.Roles = c.Roles

	ts.Databases, ps.DatabaseStats, err = postgres.GetDatabases(ctx, connection)
	if err != nil {
		logger.PrintError("Error collecting pg_databases: %s", err)
		return
	}

	bufferCacheReady := make(chan state.BufferCache)
	go func() {
		postgres.GetBufferCache(ctx, c, server, opts, bufferCacheReady)
	}()

	ps.LastStatementStatsAt = time.Now()
	err = postgres.SetQueryTextStatementTimeout(ctx, connection, logger, server)
	if err != nil {
		logger.PrintError("Error setting query text timeout: %s", err)
		return
	}
	ts.Statements, ts.StatementTexts, ps.StatementStats, err = postgres.GetStatements(ctx, c, connection, true)
	if err != nil {
		// Despite query performance data being an essential part of pganalyze, there are
		// situations where it may not be available (or it timed out), so treat it as a
		// non-fatal error, and continue snapshot collection.
		//
		// Importantly this also make sure that we may execute a pg_stat_statements_reset
		// (if configured) despite pg_stat_statements data retrieval failing, allowing
		// recovery from situations where the query text file got too large.
		//
		// Note that we do log it as an error, which gets added automatically to the snapshot's
		// CollectorErrors information.
		logger.PrintError("Error collecting pg_stat_statements: %s", err)
		var e *pq.Error
		if errors.As(err, &e) && e.Code == "55000" && opts.TestRun { // object_not_in_prerequisite_state
			shared_preload_libraries, _ := postgres.GetPostgresSetting(ctx, connection, "shared_preload_libraries")
			logger.PrintInfo("HINT - Current shared_preload_libraries setting: '%s'. Your Postgres server may need to be restarted for changes to take effect.", shared_preload_libraries)
			server.SelfTest.HintCollectionAspect(state.CollectionAspectPgStatStatements, "Current shared_preload_libraries setting: '%s'. Your Postgres server may need to be restarted for changes to take effect.", shared_preload_libraries)
		}
		err = nil
	} else {
		// Only collect plan stats when we successfully collected query stats
		ts.Plans, ps.PlanStats, err = postgres.GetPlans(ctx, c, connection, true)
		if err != nil {
			// Accept this as a non-fatal issue as this is not a critical stats (at least for now)
			logger.PrintError("Skipping query plan statistics, due to error: %s", err)
			err = nil
		}
	}
	err = postgres.SetDefaultStatementTimeout(ctx, connection, logger, server)
	if err != nil {
		logger.PrintError("Error setting statement timeout: %s", err)
		return
	}

	ps.StatementResetCounter = server.PrevState.StatementResetCounter + 1
	config := server.Grant.Load().Config
	if config.Features.StatementResetFrequency != 0 && ps.StatementResetCounter >= int(config.Features.StatementResetFrequency) {
		ps.StatementResetCounter = 0
		err = postgres.ResetStatements(ctx, c, connection)
		if err != nil {
			logger.PrintError("Error calling pg_stat_statements_reset() as requested: %s", err)
			err = nil
		} else {
			logger.PrintInfo("Successfully called pg_stat_statements_reset() for all queries, next reset in %d hours", config.Features.StatementResetFrequency/scheduler.FullSnapshotsPerHour)
			_, _, ts.ResetStatementStats, err = postgres.GetStatements(ctx, c, connection, false)
			if err != nil {
				logger.PrintError("Error collecting pg_stat_statements after reset: %s", err)
				err = nil
			}
		}
	}

	if opts.CollectPostgresSettings {
		ts.Settings, err = postgres.GetSettings(ctx, connection)
		if err != nil {
			logger.PrintError("Error collecting config settings: %s", err)
			return
		}
	}

	ts.Replication, err = postgres.GetReplication(ctx, c, connection)
	if err != nil {
		// We intentionally accept this as a non-fatal issue (at least for now), because we've historically
		// had issues make this work reliably
		logger.PrintWarning("Skipping replication statistics, due to error: %s", err)
		err = nil
	}

	ps, ts, err = postgres.GetServerStats(ctx, c, connection, ps, ts)
	if err != nil {
		logger.PrintError("Error collecting Postgres server statistics: %s", err)
		return
	}

	ts.BackendCounts, err = postgres.GetBackendCounts(ctx, c, connection)
	if err != nil {
		logger.PrintError("Error collecting backend counts: %s", err)
		return
	}

	// CollectAllSchemas relies on GetBufferCache to access the filenode OIDs before that data is discarded
	select {
	case <-ctx.Done():
	case ts.BufferCache = <-bufferCacheReady:
	}

	ps, ts, err = postgres.CollectAllSchemas(ctx, c, server, ps, ts)
	if err != nil {
		logger.PrintError("Error collecting schema information: %s", err)
		return
	}

	if server.Config.IgnoreTablePattern != "" {
		var filteredRelations []state.PostgresRelation
		patterns := strings.Split(server.Config.IgnoreTablePattern, ",")
		for _, relation := range ps.Relations {
			var matched bool
			for _, pattern := range patterns {
				matched, _ = filepath.Match(pattern, relation.SchemaName+"."+relation.RelationName)
				if matched {
					break
				}
			}
			if !matched {
				filteredRelations = append(filteredRelations, relation)
			}
		}
		ps.Relations = filteredRelations
	}

	if opts.CollectSystemInformation {
		ps.System = system.GetSystemState(ctx, server, logger, opts)
	}

	logs.SyncLogParser(server, ts.Settings)

	ps.CollectorStats = getCollectorStats()
	ts.CollectorConfig = getCollectorConfig(server.Config)
	ts.CollectorPlatform = getCollectorPlatform(server, opts, logger)

	return
}
