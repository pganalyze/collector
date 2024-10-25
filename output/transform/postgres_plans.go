package transform

import (
	"time"

	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func transformPostgresPlans(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx, statementKeyToIdx StatementKeyToIdx) snapshot.FullSnapshot {
	planInformations := []*snapshot.QueryPlanInformation{}
	planStats := []*snapshot.QueryPlanStatistic{}
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
		planInformations = append(planInformations, &snapshot.QueryPlanInformation{
			QueryIdx:         queryIdx,
			PlanId:           pKey.PlanID,
			ExplainPlan:      plan.ExplainPlan,
			PlanCapturedTime: timestamppb.New(plan.PlanCapturedTime),
		})
		stats, statsExist := diffState.PlanStats[pKey]
		if statsExist {
			planStats = append(planStats, &snapshot.QueryPlanStatistic{
				QueryIdx:          queryIdx,
				PlanId:            pKey.PlanID,
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
			})
		}
	}
	s.QueryPlanInformations = planInformations
	s.QueryPlanStatistics = planStats

	// Handle historic plan stats (similar to handling historic statement stats)
	for timeKey, diffStats := range transientState.HistoricPlanStats {
		// Ignore any data older than an hour, as a safety measure in case of many
		// failed full snapshot runs (which don't reset state)
		if time.Since(timeKey.CollectedAt).Hours() >= 1 {
			continue
		}

		h := snapshot.HistoricQueryPlanStatistics{}
		h.CollectedAt = timestamppb.New(timeKey.CollectedAt)
		h.CollectedIntervalSecs = timeKey.CollectedIntervalSecs

		for pKey, stats := range diffStats {
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
			planStat := &snapshot.QueryPlanStatistic{
				QueryIdx:          queryIdx,
				PlanId:            pKey.PlanID,
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
			h.Statistics = append(h.Statistics, planStat)
		}
		s.HistoricQueryPlanStatistics = append(s.HistoricQueryPlanStatistics, &h)
	}

	return s
}
