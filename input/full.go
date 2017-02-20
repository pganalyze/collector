package input

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// CollectFull - Collects a "full" snapshot of all data we need on a regular interval
func CollectFull(server state.Server, connection *sql.DB, collectionOpts state.CollectionOpts, logger *util.Logger) (ps state.PersistedState, ts state.TransientState, err error) {
	var explainInputs []state.PostgresExplainInput

	ps.CollectedAt = time.Now()

	ps.Version, err = postgres.GetPostgresVersion(logger, connection)
	if err != nil {
		logger.PrintError("Error collecting Postgres Version")
		return
	}

	if ps.Version.Numeric < state.MinRequiredPostgresVersion {
		err = fmt.Errorf("Error: Your PostgreSQL server version (%s) is too old, 9.2 or newer is required.", ps.Version.Short)
		return
	}

	ps.Roles, err = postgres.GetRoles(logger, connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_roles")
		return
	}

	ps.Databases, err = postgres.GetDatabases(logger, connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_databases")
		return
	}

	ps.Backends, err = postgres.GetBackends(logger, connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_activity")
		return
	}

	ts.Statements, ps.StatementStats, err = postgres.GetStatements(logger, connection, ps.Version)
	if err != nil {
		logger.PrintError("Error collecting pg_stat_statements")
		return
	}

	if collectionOpts.CollectPostgresSettings {
		ps.Settings, err = postgres.GetSettings(connection, ps.Version)
		if err != nil {
			logger.PrintError("Error collecting config settings")
			return
		}
	}

	ps = collectAllSchemas(server, collectionOpts, logger, ps)

	if collectionOpts.CollectSystemInformation {
		ps.System = system.GetSystemState(server.Config, logger)
	}

	if collectionOpts.CollectLogs {
		ps.Logs, explainInputs = system.GetLogLines(server.Config)

		if collectionOpts.CollectExplain {
			ps.Explains = postgres.RunExplain(connection, explainInputs)
		}
	}

	ps.CollectorStats = getCollectorStats()

	return
}

func collectAllSchemas(server state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, ps state.PersistedState) state.PersistedState {
	schemaDbNames := []string{}

	if server.Config.DbAllNames {
		for _, database := range ps.Databases {
			if !database.IsTemplate && database.AllowConnections {
				schemaDbNames = append(schemaDbNames, database.Name)
			}
		}
	} else {
		schemaDbNames = append(schemaDbNames, server.Config.DbName)
		schemaDbNames = append(schemaDbNames, server.Config.DbExtraNames...)
	}

	ps.RelationStats = make(state.PostgresRelationStatsMap)
	ps.IndexStats = make(state.PostgresIndexStatsMap)

	for _, dbName := range schemaDbNames {
		schemaConnection, err := postgres.EstablishConnection(server, logger, collectionOpts, dbName)
		if err != nil {
			logger.PrintVerbose("Failed to connect to database %s to retrieve schema: %s", dbName, err)
			continue
		}

		databaseOid, err := postgres.CurrentDatabaseOid(schemaConnection)
		if err != nil {
			logger.PrintError("Error getting OID of database %s")
			schemaConnection.Close()
			continue
		}

		ps = collectSchemaData(collectionOpts, logger, ps.Version, schemaConnection, ps, databaseOid)
		ps.DatabaseOidsWithLocalCatalog = append(ps.DatabaseOidsWithLocalCatalog, databaseOid)

		schemaConnection.Close()
	}

	return ps
}

func collectSchemaData(collectionOpts state.CollectionOpts, logger *util.Logger, postgresVersion state.PostgresVersion, db *sql.DB, ps state.PersistedState, databaseOid state.Oid) state.PersistedState {
	if collectionOpts.CollectPostgresRelations {
		newRelations, err := postgres.GetRelations(db, postgresVersion, databaseOid)
		if err != nil {
			logger.PrintError("Error collecting relation/index information: %s", err)
			return ps
		}
		ps.Relations = append(ps.Relations, newRelations...)

		newRelationStats, err := postgres.GetRelationStats(db, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting relation stats: %s", err)
			return ps
		}
		for k, v := range newRelationStats {
			ps.RelationStats[k] = v
		}

		newIndexStats, err := postgres.GetIndexStats(db, postgresVersion)
		if err != nil {
			logger.PrintError("Error collecting index stats: %s", err)
			return ps
		}
		for k, v := range newIndexStats {
			ps.IndexStats[k] = v
		}
	}

	if collectionOpts.CollectPostgresFunctions {
		newFunctions, err := postgres.GetFunctions(db, postgresVersion, databaseOid)
		if err != nil {
			logger.PrintError("Error collecting stored procedures")
			return ps
		}
		ps.Functions = append(ps.Functions, newFunctions...)
	}

	return ps
}
