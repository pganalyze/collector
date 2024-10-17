package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func transformPostgresPlans(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx, statementKeyToIdx StatementKeyToIdx) snapshot.FullSnapshot {
	queryPlans := make(map[int32][]*snapshot.QueryPlan)
	for pKey, plan := range transientState.Plans {
		sKey := postgresStatementKey{
			databaseOid: pKey.DatabaseOid,
			userOid:     pKey.UserOid,
			queryID:     pKey.QueryID,
		}
		queryIdx, exist := statementKeyToIdx[sKey]
		if !exist {
			// Corresponding statement (from pg_stat_statements) doesn't exist
			// Skip recording this
			continue
		}
		queryPlan := snapshot.QueryPlan{
			PlanId:           pKey.PlanID,
			ExplainPlan:      plan.ExplainPlan,
			PlanCapturedTime: timestamppb.New(plan.PlanCapturedTime),
		}
		stats, statsExist := diffState.PlanStats[pKey]
		if statsExist {
			queryPlan.QueryPlanStats = &snapshot.QueryPlanStatistic{
				Calls:             stats.Calls,
				TotalTime:         stats.TotalTime,
				Rows:              stats.Rows,
				SharedBlksHit:     stats.SharedBlksHit,
				SharedBlksRead:    stats.SharedBlksRead,
				SharedBlksDirtied: stats.SharedBlksDirtied,
				SharedBlksWritten: stats.SharedBlksWritten,
				LocalBlksHit:      stats.LocalBlksHit,
				LocalBlksRead:     stats.LocalBlksRead,
				LocalBlksDirtied:  stats.LocalBlksDirtied,
				LocalBlksWritten:  stats.LocalBlksWritten,
				TempBlksRead:      stats.TempBlksRead,
				TempBlksWritten:   stats.TempBlksWritten,
				BlkReadTime:       stats.BlkReadTime,
				BlkWriteTime:      stats.BlkWriteTime,
			}
		}
		plans, exist := queryPlans[queryIdx]
		if exist {
			plans = append(plans, &queryPlan)
		} else {
			queryPlans[queryIdx] = []*snapshot.QueryPlan{
				&queryPlan,
			}
		}
	}
	for queryIdx, plans := range queryPlans {
		planInformation := snapshot.QueryPlanInformation{
			QueryIdx:   queryIdx,
			QueryPlans: plans,
		}
		s.QueryPlanInformations = append(s.QueryPlanInformations, &planInformation)
	}

	return s
}
