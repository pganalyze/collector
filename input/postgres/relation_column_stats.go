package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pganalyze/collector/selftest"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const columnStatsSQL = `
SELECT schemaname, tablename, attname, inherited, null_frac, avg_width, n_distinct, correlation
  FROM %s
 WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
`

func GetColumnStats(ctx context.Context, logger *util.Logger, db *sql.DB, globalCollectionOpts state.CollectionOpts, systemType string, dbName string, server *state.Server) (state.PostgresColumnStatsMap, error) {
	var sourceTable string

	if StatsHelperExists(ctx, db, "get_column_stats") {
		logger.PrintVerbose("Found pganalyze.get_column_stats() stats helper")
		sourceTable = "pganalyze.get_column_stats()"
		server.SelfTest.MarkDbCollectionAspectOk(dbName, state.CollectionAspectColumnStats)
	} else {
		if systemType != "heroku" && !connectedAsSuperUser(ctx, db, systemType) && globalCollectionOpts.TestRun {
			server.SelfTest.MarkDbCollectionAspectError(dbName, state.CollectionAspectColumnStats, "monitoring helper function pganalyze.get_column_stats not found")
			server.SelfTest.HintDbCollectionAspect(dbName, state.CollectionAspectColumnStats, "Limited access to table column statistics detected. Please set up"+
				" the monitoring helper function pganalyze.get_column_stats (%s)"+
				" or connect as superuser to get column statistics for all tables.", selftest.URLPrinter.Sprint("https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user"))
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
