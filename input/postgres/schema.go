package postgres

import (
	"database/sql"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func CollectAllSchemas(server state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, ps state.PersistedState, ts state.TransientState, systemType string) (state.PersistedState, state.TransientState) {
	schemaDbNames := []string{}

	if server.Config.DbAllNames {
		for _, database := range ts.Databases {
			if !database.IsTemplate && database.AllowConnections && !(systemType == "amazon_rds" && database.Name == "rdsadmin") {
				schemaDbNames = append(schemaDbNames, database.Name)
			}
		}
	} else {
		schemaDbNames = append(schemaDbNames, server.Config.DbName)
		schemaDbNames = append(schemaDbNames, server.Config.DbExtraNames...)
	}

	ps.Relations = []state.PostgresRelation{}
	ps.RelationStats = make(state.PostgresRelationStatsMap)
	ps.IndexStats = make(state.PostgresIndexStatsMap)
	ps.Functions = []state.PostgresFunction{}

	for _, dbName := range schemaDbNames {
		schemaConnection, err := EstablishConnection(server, logger, collectionOpts, dbName)
		if err != nil {
			logger.PrintVerbose("Failed to connect to database %s to retrieve schema: %s", dbName, err)
			continue
		}

		databaseOid, err := CurrentDatabaseOid(schemaConnection)
		if err != nil {
			logger.PrintError("Error getting OID of database %s", dbName)
			schemaConnection.Close()
			continue
		}

		ps = collectSchemaData(collectionOpts, logger, schemaConnection, ps, databaseOid, dbName, ts.Version, server.Config.IgnoreSchemaRegexp)
		ts.DatabaseOidsWithLocalCatalog = append(ts.DatabaseOidsWithLocalCatalog, databaseOid)

		schemaConnection.Close()
	}
	if relCount := len(ps.Relations); relCount > 5000 {
		logger.PrintWarning("Too many tables: got %d, but only 5000 can be monitored per server; use ignore_schema_regexp config setting to filter", relCount)
	}

	return ps, ts
}

func collectSchemaData(collectionOpts state.CollectionOpts, logger *util.Logger, db *sql.DB, ps state.PersistedState, databaseOid state.Oid, databaseName string, postgresVersion state.PostgresVersion, ignoreRegexp string) state.PersistedState {
	if collectionOpts.CollectPostgresRelations {
		newRelations, err := GetRelations(db, postgresVersion, databaseOid, ignoreRegexp)
		if err != nil {
			logger.PrintWarning("Skipping table/index data for database \"%s\", due to error: %s", databaseName, err)
			return ps
		}
		ps.Relations = append(ps.Relations, newRelations...)

		newRelationStats, err := GetRelationStats(db, postgresVersion, ignoreRegexp)
		if err != nil {
			logger.PrintWarning("Skipping table statistics for database \"%s\", due to error: %s", databaseName, err)
			return ps
		}
		for k, v := range newRelationStats {
			ps.RelationStats[k] = v
		}

		newIndexStats, err := GetIndexStats(db, postgresVersion, ignoreRegexp)
		if err != nil {
			logger.PrintWarning("Skipping index statistics for database \"%s\", due to error: %s", databaseName, err)
			return ps
		}
		for k, v := range newIndexStats {
			ps.IndexStats[k] = v
		}
	}

	if collectionOpts.CollectPostgresFunctions {
		newFunctions, err := GetFunctions(db, postgresVersion, databaseOid, ignoreRegexp)
		if err != nil {
			logger.PrintWarning("Skipping stored procedure data for database \"%s\", due to error: %s", databaseName, err)
			return ps
		}
		ps.Functions = append(ps.Functions, newFunctions...)
	}

	return ps
}
