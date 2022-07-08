package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const defaultSchemaTableLimit = 5000

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
		psNext, tsNext, databaseOid, err := collectOneSchema(server, collectionOpts, logger, ps, ts, ts.Version, systemType, dbName)
		if err != nil {
			warning := "Failed to collect schema metadata for database %s: %s"
			if collectionOpts.TestRun {
				logger.PrintWarning(warning, dbName, err)
			} else {
				logger.PrintVerbose(warning, dbName, err)
			}
			continue
		}
		ps = psNext
		ts = tsNext
		ts.DatabaseOidsWithLocalCatalog = append(ts.DatabaseOidsWithLocalCatalog, databaseOid)
	}
	schemaTableLimit := server.Grant.Config.SchemaTableLimit
	if schemaTableLimit == 0 {
		schemaTableLimit = defaultSchemaTableLimit
	}
	if relCount := len(ps.Relations); relCount > schemaTableLimit {
		logger.PrintWarning("Too many tables: got %d, but only %d can be monitored per server; schema information will not be sent; learn more at https://pganalyze.com/docs/collector/settings#schema-filter-settings", relCount, schemaTableLimit)
	}

	return ps, ts
}

func collectOneSchema(server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, ps state.PersistedState, ts state.TransientState, postgresVersion state.PostgresVersion, systemType string, dbName string) (psOut state.PersistedState, tsOut state.TransientState, databaseOid state.Oid, err error) {
	schemaConnection, err := EstablishConnection(server, logger, collectionOpts, dbName)
	if err != nil {
		return ps, ts, 0, fmt.Errorf("error connecting: %s", err)
	}
	defer schemaConnection.Close()

	databaseOid, err = CurrentDatabaseOid(schemaConnection)
	if err != nil {
		return ps, ts, 0, fmt.Errorf("error getting database OID: %s", err)
	}

	ps.SchemaStats[databaseOid] = &state.SchemaStats{
		RelationStats: make(state.PostgresRelationStatsMap),
		IndexStats:    make(state.PostgresIndexStatsMap),
		ColumnStats:   make(state.PostgresColumnStatsMap),
	}

	psOut, tsOut, err = collectSchemaData(collectionOpts, logger, schemaConnection, ps, ts, databaseOid, postgresVersion, server.Config.IgnoreSchemaRegexp, systemType, dbName)
	if err != nil {
		return ps, ts, 0, err
	}

	return psOut, tsOut, databaseOid, nil
}

func collectSchemaData(collectionOpts state.CollectionOpts, logger *util.Logger, db *sql.DB, ps state.PersistedState, ts state.TransientState, databaseOid state.Oid, postgresVersion state.PostgresVersion, ignoreRegexp string, systemType string, dbName string) (state.PersistedState, state.TransientState, error) {
	if collectionOpts.CollectPostgresRelations {
		newRelations, err := GetRelations(db, postgresVersion, databaseOid, ignoreRegexp)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting table/index metadata: %s", err)
		}
		ps.Relations = append(ps.Relations, newRelations...)

		newRelationStats, err := GetRelationStats(db, postgresVersion, ignoreRegexp)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting table statistics: %s", err)
		}
		for k, v := range newRelationStats {
			ps.SchemaStats[databaseOid].RelationStats[k] = v
		}

		newIndexStats, err := GetIndexStats(db, postgresVersion, ignoreRegexp)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting index statistics: %s", err)
		}
		for k, v := range newIndexStats {
			ps.SchemaStats[databaseOid].IndexStats[k] = v
		}

		newColumnStats, err := GetColumnStats(logger, db, collectionOpts, systemType, dbName)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting column statistics: %s", err)
		}
		for k, v := range newColumnStats {
			ps.SchemaStats[databaseOid].ColumnStats[k] = v
		}
	}

	if collectionOpts.CollectPostgresFunctions {
		newFunctions, err := GetFunctions(db, postgresVersion, databaseOid, ignoreRegexp)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting stored procedure metadata: %s", err)
		}
		ps.Functions = append(ps.Functions, newFunctions...)
	}

	newExtensions, err := GetExtensions(db, databaseOid)
	if err != nil {
		return ps, ts, fmt.Errorf("error collecting extension information: %s", err)
	}
	ts.Extensions = append(ts.Extensions, newExtensions...)

	newTypes, err := GetTypes(db, postgresVersion, databaseOid)
	if err != nil {
		return ps, ts, fmt.Errorf("error collecting custom types: %s", err)
	}
	ts.Types = append(ts.Types, newTypes...)

	return ps, ts, nil
}
