package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const extendedStatisticsSQLExprsField = "null"
const extendedStatisticsSQLpg14ExprsField = "pg_get_statisticsobjdef_expressions(s.oid)::text[]"
const extendedStatisticsSQLInheritedField = "null"
const extendedStatisticsSQLpg15InheritedField = "sd.inherited"

const extendedStatisticsSQL = `
SELECT c.oid,
	   n.nspname,
	   s.stxname,
	   (SELECT array_agg(k) FROM unnest(s.stxkeys) k) stxkeys,
	   COALESCE(%s, ARRAY[]::text[]) exprs,
	   s.stxkind,
	   %s,
	   sd.n_distinct,
	   sd.dependencies
  FROM pg_catalog.pg_statistic_ext s
  JOIN pg_class c ON (s.stxrelid = c.oid)
  JOIN pg_namespace n ON (s.stxnamespace = n.oid)
  LEFT JOIN %s sd ON (sd.statistics_schemaname = n.nspname AND sd.statistics_name = s.stxname)
 WHERE ($1 = '' OR (n.nspname || '.' || c.relname) !~* $1)
`

func GetRelationStatsExtended(ctx context.Context, logger *util.Logger, db *sql.DB, postgresVersion state.PostgresVersion, server *state.Server, globalCollectionOpts state.CollectionOpts, systemType string, dbName string) (state.PostgresRelationStatsExtendedMap, error) {
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

	if StatsHelperExists(ctx, db, "get_relation_stats_ext") {
		logger.PrintVerbose("Found pganalyze.get_relation_stats_ext() stats helper")
		server.SelfTest.MarkDbCollectionAspectOk(dbName, state.CollectionAspectExtendedStats)
		sourceTable = "pganalyze.get_relation_stats_ext()"
	} else {
		if systemType != "heroku" && !connectedAsSuperUser(ctx, db, systemType) && globalCollectionOpts.TestRun {
			server.SelfTest.MarkDbCollectionAspectError(dbName, state.CollectionAspectExtendedStats, "monitoring helper function pganalyze.get_relation_stats_ext not found")
			logger.PrintInfo("Warning: Limited access to extended table statistics detected in database %s. Please set up"+
				" the monitoring helper function pganalyze.get_relation_stats_ext (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)"+
				" or connect as superuser, to get extended statistics for all tables.", dbName)
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

	var statsMap = make(state.PostgresRelationStatsExtendedMap)

	for rows.Next() {
		var tableOid state.Oid
		var s state.PostgresRelationStatsExtended

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
