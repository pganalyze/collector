package input

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// CollectFull - Collects a "full" snapshot of all data we need on a regular interval
func CollectFull(server *state.Server, connection *sql.DB, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (ps state.PersistedState, ts state.TransientState, err error) {
	systemType := server.Config.SystemType
	ps.CollectedAt = time.Now()

	ts.Version, err = postgres.GetPostgresVersion(logger, connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	if ts.Version.Numeric < state.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.3 or newer is required.", ts.Version.Short)
		return
	}

	ts.Roles, err = postgres.GetRoles(logger, connection, ts.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_roles")
		return
	}

	ts.Databases, ps.DatabaseStats, err = postgres.GetDatabases(logger, connection, ts.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_databases")
		return
	}

	ps.LastStatementStatsAt = time.Now()
	postgres.SetQueryTextStatementTimeout(connection, logger, server)
	ts.Statements, ts.StatementTexts, ps.StatementStats, err = postgres.GetStatements(server, logger, connection, globalCollectionOpts, ts.Version, true, systemType)
	postgres.SetDefaultStatementTimeout(connection, logger, server)
	if err != nil {
		err = fmt.Errorf("Error collecting pg_stat_statements: %s", err)
		return
	}

	ps.StatementResetCounter = server.PrevState.StatementResetCounter + 1
	if server.Grant.Config.Features.StatementResetFrequency != 0 && ps.StatementResetCounter >= server.Grant.Config.Features.StatementResetFrequency {
		ps.StatementResetCounter = 0
		err = postgres.ResetStatements(logger, connection, systemType)
		if err != nil {
			logger.PrintError("Error calling pg_stat_statements_reset() as requested: %s", err)
			return
		}
		_, _, ts.ResetStatementStats, err = postgres.GetStatements(server, logger, connection, globalCollectionOpts, ts.Version, false, systemType)
		if err != nil {
			err = fmt.Errorf("Error collecting pg_stat_statements: %s", err)
			return
		}
	}

	if globalCollectionOpts.CollectPostgresSettings {
		ts.Settings, err = postgres.GetSettings(connection)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	ts.Replication, err = postgres.GetReplication(logger, connection, ts.Version, systemType)
	if err != nil {
		// We intentionally accept this as a non-fatal issue (at least for now), because we've historically
		// had issues make this work reliably
		logger.PrintWarning("Skipping replication statistics, due to error: %s", err)
		err = nil
	}

	ts.ServerStats, ps.ServerIoStats, err = postgres.GetServerStats(logger, connection, ts.Version, systemType)
	if err != nil {
		logger.PrintError("Error collecting Postgres server statistics: %s", err)
		return
	}

	ts.BackendCounts, err = postgres.GetBackendCounts(logger, connection, ts.Version, server.Config.SystemType)
	if err != nil {
		logger.PrintError("Error collecting backend counts: %s", err)
		return
	}

	ps, ts = postgres.CollectAllSchemas(server, globalCollectionOpts, logger, ps, ts, systemType)

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

	if globalCollectionOpts.CollectSystemInformation {
		ps.System = system.GetSystemState(server.Config, logger)
	}

	ps.CollectorStats = getCollectorStats()
	ts.CollectorConfig = getCollectorConfig(server.Config)
	ts.CollectorPlatform = getCollectorPlatform(globalCollectionOpts, logger)

	return
}
