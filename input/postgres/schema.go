package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
)

const defaultSchemaTableLimit = 5000

// Since schema data collection is a special case that can fail when a server has
// a lot of databases (e.g. multi-tenant use case), we explicitly have a shorter
// timeout than a full collection interval (10 minutes)
const schemaCollectionTimeout = 8 * time.Minute

func isCloudInternalDatabase(systemType string, databaseName string) bool {
	if systemType == "amazon_rds" {
		return databaseName == "rdsadmin"
	}
	if systemType == "azure_database" {
		return databaseName == "azure_maintenance"
	}
	if systemType == "google_cloudsql" {
		return databaseName == "cloudsqladmin"
	}
	return false
}

func GetDatabasesToCollect(config config.ServerConfig, databases []state.PostgresDatabase) []string {
	schemaDbNames := []string{}
	if config.DbAllNames {
		for _, database := range databases {
			if !database.IsTemplate && database.AllowConnections && !isCloudInternalDatabase(config.SystemType, database.Name) {
				schemaDbNames = append(schemaDbNames, database.Name)
			}
		}
	} else {
		schemaDbNames = append(schemaDbNames, config.DbName)
		schemaDbNames = append(schemaDbNames, config.DbExtraNames...)
	}
	return schemaDbNames
}

func CollectAllSchemas(ctx context.Context, c *Collection, server *state.Server, ps state.PersistedState, ts state.TransientState) (state.PersistedState, state.TransientState, error) {
	ctxSchema, cancel := context.WithTimeout(ctx, schemaCollectionTimeout)
	defer cancel()

	ps.Relations = []state.PostgresRelation{}

	ps.SchemaStats = make(map[state.Oid]*state.SchemaStats)
	ps.Functions = []state.PostgresFunction{}

	collected := make(map[string]bool)
	for _, dbName := range GetDatabasesToCollect(server.Config, ts.Databases) {
		if _, ok := collected[dbName]; ok {
			continue
		}
		c.SelfTest.MarkMonitoredDb(dbName)

		collected[dbName] = true
		psNext, tsNext, databaseOid, err := collectOneSchema(ctxSchema, c, server, ps, ts, dbName)
		if err != nil {
			// If the outer context failed, return an error to the caller
			if ctx.Err() != nil {
				c.SelfTest.MarkRemainingDbCollectionAspectError(state.CollectionAspectSchema, err.Error())
				return ps, ts, err
			}
			// If the schema context failed, stop doing any further collection.
			// We avoid returning an error in this case to allow other collector
			// functions to report their data, and send any schema information
			// we already collected.
			if ctxSchema.Err() != nil {
				c.Logger.PrintWarning("Failed to collect schema metadata for database %s and all remaining databases: %s", dbName, err)
				c.SelfTest.MarkRemainingDbCollectionAspectError(state.CollectionAspectSchema, err.Error())
				return ps, ts, nil
			}
			warning := "Failed to collect schema metadata for database %s: %s"
			c.SelfTest.MarkDbCollectionAspectError(dbName, state.CollectionAspectSchema, err.Error())
			if c.GlobalOpts.TestRun {
				c.Logger.PrintWarning(warning, dbName, err)
			} else {
				c.Logger.PrintVerbose(warning, dbName, err)
			}

			continue
		}
		ps = psNext
		ts = tsNext
		ts.DatabaseOidsWithLocalCatalog = append(ts.DatabaseOidsWithLocalCatalog, databaseOid)
		c.SelfTest.MarkDbCollectionAspectOk(dbName, state.CollectionAspectSchema)
	}
	schemaTableLimit := int(server.Grant.Load().Config.SchemaTableLimit)
	if schemaTableLimit == 0 {
		schemaTableLimit = defaultSchemaTableLimit
	}
	if relCount := len(ps.Relations); relCount > schemaTableLimit {
		// technically this is a server problem, but we can report it at the database level
		if c.GlobalOpts.TestRun {
			for _, dbName := range c.SelfTest.MonitoredDbs {
				c.SelfTest.MarkDbCollectionAspectError(dbName, state.CollectionAspectSchema, "too many total tables")
				c.SelfTest.HintDbCollectionAspect(dbName, state.CollectionAspectSchema, "Too many total tables: got %d, but only %d can be monitored per server; schema information will not be sent; learn more at %s", relCount, schemaTableLimit, selftest.URLPrinter.Sprint("https://pganalyze.com/docs/collector/settings#schema-filter-settings"))
			}
		}
		c.Logger.PrintWarning("Too many tables: got %d, but only %d can be monitored per server; schema information will not be sent; learn more at https://pganalyze.com/docs/collector/settings#schema-filter-settings", relCount, schemaTableLimit)
	}

	return ps, ts, nil
}

func collectOneSchema(ctx context.Context, c *Collection, server *state.Server, ps state.PersistedState, ts state.TransientState, dbName string) (psOut state.PersistedState, tsOut state.TransientState, databaseOid state.Oid, err error) {
	schemaConnection, err := EstablishConnection(ctx, server, c.Logger, c.GlobalOpts, dbName)
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

	psOut, tsOut, err = collectSchemaData(ctx, c, schemaConnection, ps, ts, databaseOid, server, dbName)
	if err != nil {
		return ps, ts, 0, err
	}

	return psOut, tsOut, databaseOid, nil
}

func collectSchemaData(ctx context.Context, c *Collection, db *sql.DB, ps state.PersistedState, ts state.TransientState, databaseOid state.Oid, server *state.Server, dbName string) (state.PersistedState, state.TransientState, error) {
	newFunctions, err := GetFunctions(ctx, c.Logger, db, ts.Version, databaseOid, server.Config.IgnoreSchemaRegexp, false)
	if err != nil {
		return ps, ts, fmt.Errorf("error collecting stored procedure metadata: %s", err)
	}
	ps.Functions = append(ps.Functions, newFunctions...)

	c = c.ForCurrentDatabase(newFunctions)

	if c.GlobalOpts.CollectPostgresRelations {
		newRelations, err := GetRelations(ctx, c, db, databaseOid)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting table/index metadata: %s", err)
		}
		ps.Relations = append(ps.Relations, newRelations...)

		newRelationStats, err := GetRelationStats(ctx, c, db, databaseOid, ts)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting table statistics: %s", err)
		}
		for k, v := range newRelationStats {
			ps.SchemaStats[databaseOid].RelationStats[k] = v
		}

		newIndexStats, err := GetIndexStats(ctx, c, db, databaseOid, ts)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting index statistics: %s", err)
		}
		for k, v := range newIndexStats {
			ps.SchemaStats[databaseOid].IndexStats[k] = v
		}

		newColumnStats, err := GetColumnStats(ctx, c, db, dbName)
		if err != nil {
			return ps, ts, fmt.Errorf("error collecting column statistics: %s", err)
		}
		for k, v := range newColumnStats {
			ps.SchemaStats[databaseOid].ColumnStats[k] = v
		}

		if c.PostgresVersion.Numeric >= state.PostgresVersion12 {
			newRelationStatsExtended, err := GetRelationStatsExtended(ctx, c, db, dbName)
			if err != nil {
				return ps, ts, fmt.Errorf("error collecting extended relation statistics: %s", err)
			}
			for k, v := range newRelationStatsExtended {
				ps.SchemaStats[databaseOid].RelationStatsExtended[k] = v
			}
		}
	}

	newExtensions, err := GetExtensions(ctx, db, databaseOid)
	if err != nil {
		return ps, ts, fmt.Errorf("error collecting extension information: %s", err)
	}
	ts.Extensions = append(ts.Extensions, newExtensions...)

	newTypes, err := GetTypes(ctx, c, db, databaseOid)
	if err != nil {
		return ps, ts, fmt.Errorf("error collecting custom types: %s", err)
	}
	ts.Types = append(ts.Types, newTypes...)

	return ps, ts, nil
}
