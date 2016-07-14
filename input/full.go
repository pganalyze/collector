package input

import (
	"fmt"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// CollectFull - Collects a "full" snapshot of all data we need on a regular interval
func CollectFull(server state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) (ps state.PersistedState, ts state.TransientState, err error) {
	var explainInputs []state.PostgresExplainInput

	ps.Version, err = postgres.GetPostgresVersion(logger, server.Connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	ps.DataDirectory, err = postgres.GetDataDirectory(logger, server.Connection)
	if err != nil {
		logger.PrintVerbose("Could not determine data directory location")
	}

	currentDatabaseOid, err := postgres.CurrentDatabaseOid(server.Connection)
	if err != nil {
		logger.PrintError("Error getting OID of current database")
		return
	}
	ps.DatabaseOidsWithLocalCatalog = append(ps.DatabaseOidsWithLocalCatalog, currentDatabaseOid)

	if ps.Version.Numeric < state.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required.", ps.Version.Short)
		return
	}

	ps.Roles, err = postgres.GetRoles(logger, server.Connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_roles")
		return
	}

	ps.Databases, err = postgres.GetDatabases(logger, server.Connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_databases")
		return
	}

	ps.Backends, err = postgres.GetBackends(logger, server.Connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_activity")
		return
	}

	ts.Statements, ps.StatementStats, err = postgres.GetStatements(logger, server.Connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_statements")
		return
	}

	if collectionOpts.CollectPostgresRelations {
		ps.Relations, err = postgres.GetRelations(server.Connection, ps.Version, currentDatabaseOid)
		if err != nil {
			logger.PrintError("Error collecting relation/index information: %s", err)
			return
		}

		ps.RelationStats, err = postgres.GetRelationStats(server.Connection, ps.Version)
		if err != nil {
			logger.PrintError("Error collecting relation stats: %s", err)
			return
		}

		ps.IndexStats, err = postgres.GetIndexStats(server.Connection, ps.Version)
		if err != nil {
			logger.PrintError("Error collecting index stats: %s", err)
			return
		}

		// collectionOpts.CollectPostgresBloat
	}

	if collectionOpts.CollectPostgresFunctions {
		ps.Functions, err = postgres.GetFunctions(server.Connection, ps.Version, currentDatabaseOid)
		if err != nil {
			logger.PrintError("Error collecting stored procedures")
			return
		}
	}

	if collectionOpts.CollectPostgresSettings {
		ps.Settings, err = postgres.GetSettings(server.Connection, ps.Version)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	if collectionOpts.CollectSystemInformation {
		ps.System = system.GetSystemState(server.Config, logger, ps.DataDirectory)
	}

	if collectionOpts.CollectLogs {
		ps.Logs, explainInputs = system.GetLogLines(server.Config)

		if collectionOpts.CollectExplain {
			ps.Explains = postgres.RunExplain(server.Connection, explainInputs)
		}
	}

	ps.CollectorStats = getCollectorStats()

	return
}
