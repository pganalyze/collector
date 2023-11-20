package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const columnStatsSQL = `
SELECT schemaname, tablename, attname, inherited, null_frac, avg_width, n_distinct, correlation
  FROM %s
 WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
`

const extendedStatisticsSQLExprsField = "ARRAY[]::text[]"
const extendedStatisticsSQLpg14ExprsField = "COALESCE(pg_get_statisticsobjdef_expressions(s.oid)::text[], ARRAY[]::text[]) exprs"

const extendedStatisticsSQLInheritedField = "false"
const extendedStatisticsSQLpg15InheritedField = "sd.inherited"

const extendedStatisticsSQL = `
SELECT c.oid,
	   n.nspname,
	   s.stxname,
	   (SELECT array_agg(k) FROM unnest(s.stxkeys) k) stxkeys,
	   %s,
	   s.stxkind,
	   %s,
	   sd.n_distinct,
	   sd.dependencies
  FROM pg_statistic_ext s
  JOIN pg_class c ON (s.stxrelid = c.oid)
  JOIN pg_namespace n ON (s.stxnamespace = n.oid)
  LEFT JOIN %s sd ON (sd.statistics_schemaname = n.nspname AND sd.statistics_name = s.stxname)
 WHERE ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
`

func GetColumnStats(ctx context.Context, logger *util.Logger, db *sql.DB, globalCollectionOpts state.CollectionOpts, systemType string, dbName string) (state.PostgresColumnStatsMap, error) {
	var sourceTable string

	if StatsHelperExists(ctx, db, "get_column_stats") {
		logger.PrintVerbose("Found pganalyze.get_column_stats() stats helper")
		sourceTable = "pganalyze.get_column_stats()"
	} else {
		if systemType != "heroku" && !connectedAsSuperUser(ctx, db, systemType) && globalCollectionOpts.TestRun {
			logger.PrintInfo("Warning: Limited access to table column statistics detected in database %s. Please set up"+
				" the monitoring helper function pganalyze.get_column_stats (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)"+
				" or connect as superuser, to get column statistics for all tables.", dbName)
		}
		sourceTable = "pg_catalog.pg_stats"
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(columnStatsSQL, sourceTable))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statsMap = make(state.PostgresColumnStatsMap)

	for rows.Next() {
		var s state.PostgresColumnStats

		err := rows.Scan(
			&s.SchemaName, &s.TableName, &s.ColumnName, &s.Inherited, &s.NullFrac, &s.AvgWidth, &s.NDistinct, &s.Correlation)
		if err != nil {
			return nil, err
		}

		key := state.PostgresColumnStatsKey{SchemaName: s.SchemaName, TableName: s.TableName, ColumnName: s.ColumnName}
		statsMap[key] = append(statsMap[key], s)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return statsMap, nil
}

func GetColumnStatsExtended(ctx context.Context, logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, server *state.Server, globalCollectionOpts state.CollectionOpts, systemType string, dbName string) (state.PostgresColumnStatsExtendedMap, error) {
	var sourceTable string
	var exprsField string
	var inheritedField string

	if postgresVersion.Numeric >= state.PostgresVersion14 {
		exprsField = extendedStatisticsSQLpg14ExprsField
	} else {
		exprsField = extendedStatisticsSQLExprsField
	}

	if postgresVersion.Numeric >= state.PostgresVersion15 {
		inheritedField = extendedStatisticsSQLpg15InheritedField
	} else {
		inheritedField = extendedStatisticsSQLInheritedField
	}

	if StatsHelperExists(ctx, db, "get_column_stats_ext") {
		logger.PrintVerbose("Found pganalyze.get_column_stats_ext() stats helper")
		sourceTable = "pganalyze.get_column_stats_ext()"
	} else {
		if systemType != "heroku" && !connectedAsSuperUser(ctx, db, systemType) && globalCollectionOpts.TestRun {
			logger.PrintInfo("Warning: Limited access to table column statistics detected in database %s. Please set up"+
				" the monitoring helper function pganalyze.get_column_stats_ext (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)"+
				" or connect as superuser, to get column statistics for all tables.", dbName)
		}
		sourceTable = "pg_catalog.pg_stats_ext"
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(extendedStatisticsSQL, exprsField, inheritedField, sourceTable))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, server.Config.IgnoreSchemaRegexp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statsMap = make(state.PostgresColumnStatsExtendedMap)

	for rows.Next() {
		var tableOid state.Oid
		var s state.PostgresColumnStatsExtended

		err := rows.Scan(
			&tableOid, &s.StatisticsSchema, &s.StatisticsName, pq.Array(&s.Columns), pq.Array(&s.Expressions), pq.Array(&s.Kind), &s.Inherited, &s.NDistinct, &s.Dependencies)
		if err != nil {
			return nil, err
		}

		statsMap[tableOid] = append(statsMap[tableOid], s)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return statsMap, nil
}
