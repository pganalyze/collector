package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func CollectAllSchemas(server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, ps state.PersistedState, ts state.TransientState, systemType string) (state.PersistedState, state.TransientState) {
	schemaDbNames := []string{}

	if server.Config.DbAllNames {
		for _, database := range ts.Databases {
			if !database.IsTemplate && database.AllowConnections && !isCloudInternalDatabase(systemType, database.Name) {
				schemaDbNames = append(schemaDbNames, database.Name)
			}
		}
	} else {
		schemaDbNames = append(schemaDbNames, server.Config.DbName)
		schemaDbNames = append(schemaDbNames, server.Config.DbExtraNames...)
	}

	ps.Relations = []state.PostgresRelation{}

	ps.SchemaStats = make(map[state.Oid]*state.SchemaStats)
	ps.Functions = []state.PostgresFunction{}

	collected := make(map[string]bool)
	for _, dbName := range schemaDbNames {
		if _, ok := collected[dbName]; ok {
			continue
		}
		collected[dbName] = true
		psNext, databaseOid, err := collectOneSchema(server, collectionOpts, logger, ps, dbName, ts.Version, systemType)
		if err != nil {
			logger.PrintVerbose("Failed to collect schema metadata for database %s: %s", dbName, err)
			continue
		}
		ps = psNext
		ts.DatabaseOidsWithLocalCatalog = append(ts.DatabaseOidsWithLocalCatalog, databaseOid)
	}
	if relCount := len(ps.Relations); relCount > 5000 {
		logger.PrintWarning("Too many tables: got %d, but only 5000 can be monitored per server; use ignore_schema_regexp config setting to filter", relCount)
	}

	return ps, ts
}

func collectOneSchema(server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, ps state.PersistedState, dbName string, postgresVersion state.PostgresVersion, systemType string) (psOut state.PersistedState, databaseOid state.Oid, err error) {
	schemaConnection, err := EstablishConnection(server, logger, collectionOpts, dbName)
	if err != nil {
		return ps, 0, fmt.Errorf("error connecting: %s", err)
	}
	defer schemaConnection.Close()

	databaseOid, err = CurrentDatabaseOid(schemaConnection)
	if err != nil {
		return ps, 0, fmt.Errorf("error getting database OID: %s", err)
	}

	ps.SchemaStats[databaseOid] = &state.SchemaStats{
		RelationStats: make(state.PostgresRelationStatsMap),
		IndexStats:    make(state.PostgresIndexStatsMap),
		ColumnStats:   make([]state.PostgresColumnStats, 0),
	}

	psOut, err = collectSchemaData(collectionOpts, logger, schemaConnection, ps, databaseOid, postgresVersion, server.Config.IgnoreSchemaRegexp, systemType)
	if err != nil {
		return ps, 0, err
	}

	return psOut, databaseOid, nil
}

func collectSchemaData(collectionOpts state.CollectionOpts, logger *util.Logger, db *sql.DB, ps state.PersistedState, databaseOid state.Oid, postgresVersion state.PostgresVersion, ignoreRegexp string, systemType string) (state.PersistedState, error) {
	if collectionOpts.CollectPostgresRelations {
		newRelations, err := GetRelations(db, postgresVersion, databaseOid, ignoreRegexp)
		if err != nil {
			return ps, fmt.Errorf("error collecting table/index metadata: %s", err)
		}
		ps.Relations = append(ps.Relations, newRelations...)

		newRelationStats, err := GetRelationStats(db, postgresVersion, ignoreRegexp)
		if err != nil {
			return ps, fmt.Errorf("error collecting table statistics: %s", err)
		}
		for k, v := range newRelationStats {
			ps.SchemaStats[databaseOid].RelationStats[k] = v
		}

		newIndexStats, err := GetIndexStats(db, postgresVersion, ignoreRegexp)
		if err != nil {
			return ps, fmt.Errorf("error collecting index statistics: %s", err)
		}
		for k, v := range newIndexStats {
			ps.SchemaStats[databaseOid].IndexStats[k] = v
		}

		newColumnStats, err := GetColumnStats(logger, db, collectionOpts, systemType)
		if err != nil {
			return ps, fmt.Errorf("error collecting column statistics: %s", err)
		}
		ps.SchemaStats[databaseOid].ColumnStats = newColumnStats
	}

	if collectionOpts.CollectPostgresFunctions {
		newFunctions, err := GetFunctions(db, postgresVersion, databaseOid, ignoreRegexp)
		if err != nil {
			return ps, fmt.Errorf("error collecting stored procedure metadata: %s", err)
		}
		ps.Functions = append(ps.Functions, newFunctions...)
	}

	return ps, nil
}
