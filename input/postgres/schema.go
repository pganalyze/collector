package postgres

import (
	"database/sql"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func CollectAllSchemas(server state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, ps state.PersistedState) state.PersistedState {
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
			logger.PrintError("Error getting OID of database %s")
			schemaConnection.Close()
			continue
		}

		ps = collectSchemaData(collectionOpts, logger, schemaConnection, ps, databaseOid)
		ps.DatabaseOidsWithLocalCatalog = append(ps.DatabaseOidsWithLocalCatalog, databaseOid)

		schemaConnection.Close()
	}

	return ps
}

func collectSchemaData(collectionOpts state.CollectionOpts, logger *util.Logger, db *sql.DB, ps state.PersistedState, databaseOid state.Oid) state.PersistedState {
	if collectionOpts.CollectPostgresRelations {
		newRelations, err := GetRelations(db, ps.Version, databaseOid)
		if err != nil {
			logger.PrintError("Error collecting relation/index information: %s", err)
			return ps
		}
		ps.Relations = append(ps.Relations, newRelations...)

		newRelationStats, err := GetRelationStats(db, ps.Version)
		if err != nil {
			logger.PrintError("Error collecting relation stats: %s", err)
			return ps
		}
		for k, v := range newRelationStats {
			ps.RelationStats[k] = v
		}

		newIndexStats, err := GetIndexStats(db, ps.Version)
		if err != nil {
			logger.PrintError("Error collecting index stats: %s", err)
			return ps
		}
		for k, v := range newIndexStats {
			ps.IndexStats[k] = v
		}
	}

	if collectionOpts.CollectPostgresFunctions {
		newFunctions, err := GetFunctions(db, ps.Version, databaseOid)
		if err != nil {
			logger.PrintError("Error collecting stored procedures")
			return ps
		}
		ps.Functions = append(ps.Functions, newFunctions...)
	}

	return ps
}
