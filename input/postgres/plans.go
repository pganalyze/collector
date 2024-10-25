package postgres

import (
	"context"
	"database/sql"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// Do not query with plan_type 'no plan', as it's a query without a meaningful plan (planid=0)
// e.g. FETCH 50 IN "query-cursor_1"
const planSQL string = `
SELECT
	userid, dbid, toplevel, queryid, planid, query,
	explain_plan, plan_type, plan_captured_time,
	calls, total_exec_time, min_exec_time, max_exec_time, mean_exec_time, stddev_exec_time,
	rows, shared_blks_hit, shared_blks_read, shared_blks_dirtied, shared_blks_written,
	local_blks_hit, local_blks_read, local_blks_dirtied, local_blks_written,
	temp_blks_read, temp_blks_written,
	blk_read_time, blk_write_time
FROM
	aurora_stat_plans(true)
WHERE
	plan_type IN ('estimate', 'actual')`

// GetPlans collects query execution plans and stats
func GetPlans(ctx context.Context, server *state.Server, logger *util.Logger, db *sql.DB, globalCollectionOpts state.CollectionOpts, postgresVersion state.PostgresVersion) (state.PostgresPlanMap, state.PostgresPlanStatsMap, error) {
	var err error

	// Currently only collects this with Aurora using aurora_stat_plans function
	if !postgresVersion.IsAwsAurora {
		return nil, nil, nil
	}

	computePlanIdEnabled, err := GetPostgresSetting(ctx, "aurora_compute_plan_id", server, globalCollectionOpts, logger)
	// aurora_compute_plan_id needs to be on to use aurora_stat_plans function
	if err != nil || computePlanIdEnabled != "on" {
		if globalCollectionOpts.TestRun {
			logger.PrintInfo("Function aurora_stat_plans() is not supported or aurora_compute_plan_id is not enabled. Skip collecting query plans and stats.")
		}
		return nil, nil, nil
	}

	stmt, err := db.PrepareContext(ctx, QueryMarkerSQL+planSQL)
	if err != nil {
		return nil, nil, err
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	plans := make(state.PostgresPlanMap)
	planStats := make(state.PostgresPlanStatsMap)

	for rows.Next() {
		var key state.PostgresPlanKey
		var plan state.PostgresPlan
		var queryID null.Int
		var receivedQuery null.String
		var stats state.PostgresStatementStats

		err = rows.Scan(&key.UserOid, &key.DatabaseOid, &key.TopLevel, &queryID, &key.PlanID, &receivedQuery,
			&plan.ExplainPlan, &plan.PlanType, &plan.PlanCapturedTime,
			&stats.Calls, &stats.TotalTime, &stats.MinTime, &stats.MaxTime, &stats.MeanTime, &stats.StddevTime,
			&stats.Rows, &stats.SharedBlksHit, &stats.SharedBlksRead, &stats.SharedBlksDirtied, &stats.SharedBlksWritten,
			&stats.LocalBlksHit, &stats.LocalBlksRead, &stats.LocalBlksDirtied, &stats.LocalBlksWritten,
			&stats.TempBlksRead, &stats.TempBlksWritten, &stats.BlkReadTime, &stats.BlkWriteTime)
		if err != nil {
			return nil, nil, err
		}

		if queryID.Valid {
			key.QueryID = queryID.Int64
		} else {
			// We can't process this entry, most likely a permission problem with reading the query ID
			continue
		}

		plans[key] = plan
		planStats[key] = stats
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	return plans, planStats, nil
}
