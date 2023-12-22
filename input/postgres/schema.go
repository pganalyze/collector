package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const defaultSchemaTableLimit = 5000

// Since schema data collection is a special case that can fail when a server has
// a lot of databases (e.g. multi-tenant use case), we explicitly have a shorter
// timeout than a full collection interval (10 minutes)
const schemaCollectionTimeout = 8 * time.Minute

func CollectAllSchemas(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, ps state.PersistedState, ts state.TransientState, systemType string) (state.PersistedState, state.TransientState, error) {
	schemaDbNames := []string{}

	ctxSchema, cancel := context.WithTimeout(ctx, schemaCollectionTimeout)
	defer cancel()

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
		psNext, tsNext, databaseOid, err := collectOneSchema(ctxSchema, server, collectionOpts, logger, ps, ts, ts.Version, systemType, dbName)
		if err != nil {
			// If the outer context failed, return an error to the caller
			if ctx.Err() != nil {
				return ps, ts, err
			}
			// If the schema context failed, stop doing any further collection.
			// We avoid returning an error in this case to allow other collector
			// functions to report their data, and send any schema information
			// we already collected.
			if ctxSchema.Err() != nil {
				logger.PrintWarning("Failed to collect schema metadata for database %s and all remaining databases: %s", dbName, err)
				return ps, ts, nil
			}
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

	return ps, ts, nil
}

func collectOneSchema(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, ps state.PersistedState, ts state.TransientState, postgresVersion state.PostgresVersion, systemType string, dbName string) (psOut state.PersistedState, tsOut state.TransientState, databaseOid state.Oid, err error) {
	schemaConnection, err := EstablishConnection(ctx, server, logger, collectionOpts, dbName)
	if err != nil {
		return ps, ts, 0, fmt.Errorf("error connecting: %s", err)
	}
	defer schemaConnection.Close()

	databaseOid, err = CurrentDatabaseOid(ctx, schemaConnection)
	if err != nil {
		return ps, ts, 0, fmt.Errorf("error getting database OID: %s", err)
	}

	ps.SchemaStats[databaseOid] = &state.SchemaStats{
		RelationStats:         make(state.PostgresRelationStatsMap),
		IndexStats:            make(state.PostgresIndexStatsMap),
		ColumnStats:           make(state.PostgresColumnStatsMap),
		RelationStatsExtended: make(state.PostgresRelationStatsExtendedMap),
	}

	psOut, tsOut, err = collectSchemaData(ctx, collectionOpts, logger, schemaConnection, ps, ts, databaseOid, postgresVersion, server, systemType, dbName)
	if err != nil {
		return ps, ts, 0, err
	}

	return psOut, tsOut, databaseOid, nil
}

func collectSchemaData(ctx context.Context, collectionOpts state.CollectionOpts, logger *util.Logger, db *sql.DB, ps state.PersistedState, ts state.TransientState, databaseOid state.Oid, postgresVersion state.PostgresVersion, server *state.Server, systemType string, dbName string) (state.PersistedState, state.TransientState, error) {
	if collectionOpts.CollectPostgresRelations {
		newRelations, err := GetRelations(ctx, db, postgresVersion, databaseOid, server.Config.IgnoreSchemaRegexp)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting table/index metadata: %s", err)
		}
		ps.Relations = append(ps.Relations, newRelations...)

		newRelationStats, err := GetRelationStats(ctx, db, postgresVersion, server)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting table statistics: %s", err)
		}
		for k, v := range newRelationStats {
			ps.SchemaStats[databaseOid].RelationStats[k] = v
		}

		newIndexStats, err := GetIndexStats(ctx, db, postgresVersion, server)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting index statistics: %s", err)
		}
		for k, v := range newIndexStats {
			ps.SchemaStats[databaseOid].IndexStats[k] = v
		}

		newColumnStats, err := GetColumnStats(ctx, logger, db, collectionOpts, systemType, dbName)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting column statistics: %s", err)
		}
		for k, v := range newColumnStats {
			ps.SchemaStats[databaseOid].ColumnStats[k] = v
		}

		newRelationStatsExtended, err := GetRelationStatsExtended(ctx, logger, db, postgresVersion, server, collectionOpts, systemType, dbName)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting extended column statistics: %s", err)
		}
		for k, v := range newRelationStatsExtended {
			ps.SchemaStats[databaseOid].RelationStatsExtended[k] = v
		}
	}

	if collectionOpts.CollectPostgresFunctions {
		newFunctions, err := GetFunctions(ctx, db, postgresVersion, databaseOid, server.Config.IgnoreSchemaRegexp)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting stored procedure metadata: %s", err)
		}
		ps.Functions = append(ps.Functions, newFunctions...)
	}

	newExtensions, err := GetExtensions(ctx, db, databaseOid)
	if err != nil {
		return ps, ts, fmt.Errorf("error collecting extension information: %s", err)
	}
	ts.Extensions = append(ts.Extensions, newExtensions...)

	newTypes, err := GetTypes(ctx, db, postgresVersion, databaseOid)
	if err != nil {
		return ps, ts, fmt.Errorf("error collecting custom types: %s", err)
	}
	ts.Types = append(ts.Types, newTypes...)

	return ps, ts, nil
}
