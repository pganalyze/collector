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
//
// Note that data collection in this function is ordered in a certain way to optimize for
// both correctness (metric data capturing should be close to the "collected at" timestamp),
// whilst staying aware of the potential slow operations (query texts, schema collection).
//
// When gathering new information/metrics, add fast metric collections right before query
// text collection, and any slow operations (or those that are not a measured metric at a
// certain point) after the schema collection.
func CollectFull(ctx context.Context, server *state.Server, connection *sql.DB, opts state.CollectionOpts, logger *util.Logger) (ps state.PersistedState, ts state.TransientState, err error) {
	ps.CollectedAt = time.Now()

	bufferCacheReady := make(chan state.BufferCache)
	go func() {
		if server.Config.MaxBufferCacheMonitoringGB > 0 {
			bufferCacheReady <- postgres.GetBufferCache(ctx, server, logger, opts)
		} else {
			bufferCacheReady <- make(state.BufferCache)
		}
	}()

	systemStateReady := make(chan state.SystemState)
	go func() {
		if opts.CollectSystemInformation {
			systemStateReady <- system.GetSystemState(ctx, server, logger, opts)
		} else {
			systemStateReady <- state.SystemState{}
		}
	}()

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

	// Perform one high frequency stats collection at the exact time of the full snapshot.
	//
	// The scheduler skips the otherwise scheduled execution when the full snapshot time happens,
	// so we can run it inline here and pass its data along as part of this full snapshot.
	server.HighFreqStateMutex.Lock()
	newHighFreqState, err := CollectAndDiff1minStats(ctx, c, connection, ps.CollectedAt, server.HighFreqPrevState)
	if err != nil {
		logger.PrintError("Could not collect high frequency statistics for server: %s", err)
		err = nil
	} else {
		// Move high frequency stats to be submitted with this full snapshot. We do this early in
		// the input step (vs in the runner) to avoid additional high frequency query stats being
		// collected, whose query texts were not yet picked up with this full snapshot.
		ts.StatementStats = newHighFreqState.UnidentifiedStatementStats
		ts.PlanStats = newHighFreqState.UnidentifiedPlanStats
		ts.ServerIoStats = newHighFreqState.QueuedServerIoStats
		newHighFreqState.UnidentifiedStatementStats = make(state.HistoricStatementStatsMap)
		newHighFreqState.UnidentifiedPlanStats = make(state.HistoricPlanStatsMap)
		newHighFreqState.QueuedServerIoStats = make(state.HistoricPostgresServerIoStatsMap)
		server.HighFreqPrevState = newHighFreqState
	}
	server.HighFreqStateMutex.Unlock()

	// Collect other metrics that are fast to gather first, to avoid skewing their data points when query text collection is slow
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

	// Collect query texts (this may be slow)
	err = postgres.SetQueryTextStatementTimeout(ctx, connection, logger, server)
	if err != nil {
		logger.PrintError("Error setting query text timeout: %s", err)
		return
	}
	ts.Statements, ts.StatementTexts, err = postgres.GetStatementTexts(ctx, c, connection)
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
		// Only collect plan texts when we successfully collected query texts
		ts.Plans, _, err = postgres.GetPlans(ctx, c, connection, true)
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

	// Reset query stats and texts if needed (this must run after the query text collection)
	ps.StatementResetCounter = server.PrevState.StatementResetCounter + 1
	config := server.Grant.Load().Config
	if config.Features.StatementResetFrequency != 0 && ps.StatementResetCounter >= int(config.Features.StatementResetFrequency) {
		// Block concurrent collection of query stats, as that may see the actual Postgres-side
		// reset before we updated the struct that the collector diffs against.
		server.HighFreqStateMutex.Lock()
		ps.StatementResetCounter = 0
		err = postgres.ResetStatements(ctx, c, connection)
		if err != nil {
			logger.PrintError("Error calling pg_stat_statements_reset() as requested: %s", err)
			err = nil
		} else {
			logger.PrintInfo("Successfully called pg_stat_statements_reset() for all queries, next reset in %d hours", config.Features.StatementResetFrequency/scheduler.FullSnapshotsPerHour)

			// Make sure the next high frequency run has an empty reference point
			newHighFreqState.LastStatementStatsAt = time.Now()
			resetStatementStats, err := postgres.GetStatementStats(ctx, c, connection)
			if err != nil {
				logger.PrintError("Error collecting pg_stat_statements after reset: %s", err)
				err = nil
				newHighFreqState.StatementStats = make(state.PostgresStatementStatsMap)
			} else {
				newHighFreqState.StatementStats = resetStatementStats
			}
		}
		server.HighFreqStateMutex.Unlock()
	}

	if opts.CollectPostgresSettings {
		ts.Settings, err = postgres.GetSettings(ctx, connection)
		if err != nil {
			logger.PrintError("Error collecting config settings: %s", err)
			return
		}
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

	select {
	case <-ctx.Done():
	case ps.System = <-systemStateReady:
	}

	logs.SyncLogParser(server, ts.Settings)

	ps.CollectorStats = getCollectorStats()
	ts.CollectorConfig = getCollectorConfig(server.Config)
	ts.CollectorPlatform = getCollectorPlatform(server, opts, logger)

	return
}
