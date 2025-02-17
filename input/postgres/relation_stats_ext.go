package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
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

func GetRelationStatsExtended(ctx context.Context, c *Collection, db *sql.DB, dbName string) (state.PostgresRelationStatsExtendedMap, error) {
	var sourceTable string
	var exprsField string
	var inheritedField string

	if c.PostgresVersion.Numeric >= state.PostgresVersion14 {
		exprsField = extendedStatisticsSQLpg14ExprsField
	} else {
		exprsField = extendedStatisticsSQLExprsField
	}

	if c.PostgresVersion.Numeric >= state.PostgresVersion15 {
		inheritedField = extendedStatisticsSQLpg15InheritedField
	} else {
		inheritedField = extendedStatisticsSQLInheritedField
	}

	if StatsHelperExists(ctx, db, "get_relation_stats_ext") {
		c.Logger.PrintVerbose("Found pganalyze.get_relation_stats_ext() stats helper")
		sourceTable = "pganalyze.get_relation_stats_ext()"
		c.SelfTest.MarkDbCollectionAspectOk(dbName, state.CollectionAspectExtendedStats)
	} else {
		sourceTable = "pg_catalog.pg_stats_ext"
		if c.GlobalOpts.TestRun {
			if c.Config.SystemType == "heroku" || c.ConnectedAsSuperUser {
				c.SelfTest.MarkDbCollectionAspectOk(dbName, state.CollectionAspectExtendedStats)
			} else {
				c.SelfTest.MarkDbCollectionAspectError(dbName, state.CollectionAspectExtendedStats, "monitoring helper function pganalyze.get_relation_stats_ext not found")
				c.SelfTest.HintDbCollectionAspect(dbName, state.CollectionAspectExtendedStats, "Please set up"+
					" the monitoring helper function pganalyze.get_relation_stats_ext (%s)"+
					" or connect as superuser, to get extended statistics for all tables.", selftest.URLPrinter.Sprint("https://pganalyze.com/docs/install/troubleshooting/ext_stats_helper"))
				c.Logger.PrintInfo("Warning: Limited access to extended table statistics detected in database %s. Please set up"+
					" the monitoring helper function pganalyze.get_relation_stats_ext (https://pganalyze.com/docs/install/troubleshooting/ext_stats_helper)"+
					" or connect as superuser, to get extended statistics for all tables.", dbName)
			}
		}
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+fmt.Sprintf(extendedStatisticsSQL, exprsField, inheritedField, sourceTable))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, c.Config.IgnoreSchemaRegexp)
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
