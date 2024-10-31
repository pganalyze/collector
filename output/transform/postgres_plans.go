package transform

import (
	"time"

	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type planKey struct {
	queryIdx int32
	planID   int64
}

type planValue struct {
	plan      state.PostgresPlan
	planStats state.DiffedPostgresStatementStats
}

func groupPlans(plans state.PostgresPlanMap, statsMap state.DiffedPostgresPlanStatsMap, queryIDKeyToIdx QueryIDKeyToIdx) map[planKey]planValue {
	// We group multiple different queryIDs with the same fingerprints with one "statementKey"
	// (e.g. SELECT * FROM users WHERE name in ($1) and SELECT * FROM users WHERE name in ($1, $2) have the same fingerprints even though queryID will be different)
	// With query plans, we also group plans and plan stats with a statementKey + planID.
	groupedPlans := make(map[planKey]planValue)

	for sKey, stats := range statsMap {
		plan, exist := plans[sKey]
		if !exist {
			// When the plan associated with the stats is not found (shouldn't happen), ignore stats
			// This could happen when the plan (and stats) was there during the historical (every 1 min) collection, but is gone with the full snapshot.
			continue
		}

		qKey := queryIDKey{
			databaseOid: sKey.DatabaseOid,
			userOid:     sKey.UserOid,
			queryID:     sKey.QueryID,
		}
		queryIdx, exist := queryIDKeyToIdx[qKey]
		if !exist {
			// Corresponding statement (from pg_stat_statements) doesn't exist, ignore stats
			// This could happen when plan stats stores some query data that was already deallocated in pg_stat_statements.
			continue
		}

		key := planKey{
			queryIdx: queryIdx,
			planID:   sKey.PlanID,
		}
		value, exist := groupedPlans[key]
		if exist {
			// When there are multiple plans per group, use the most recently captured plan
			if value.plan.PlanCapturedTime.Before(plan.PlanCapturedTime) {
				groupedPlans[key] = planValue{
					plan:      plan,
					planStats: value.planStats.Add(stats),
				}
			} else {
				groupedPlans[key] = planValue{
					plan:      value.plan,
					planStats: value.planStats.Add(stats),
				}
			}
		} else {
			groupedPlans[key] = planValue{
				plan:      plan,
				planStats: stats,
			}
		}
	}
	return groupedPlans
}

func transformQueryPlanStatistic(stats state.DiffedPostgresStatementStats, idx int32) snapshot.QueryPlanStatistic {
	return snapshot.QueryPlanStatistic{
		QueryPlanIdx: idx,

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

func upsertQueryPlanReference(s *snapshot.FullSnapshot, queryIdx int32, planId int64) int32 {
	newRef := snapshot.QueryPlanReference{
		QueryIdx: queryIdx,
		PlanId:   planId,
	}

	for idx, ref := range s.QueryPlanReferences {
		if ref.QueryIdx == newRef.QueryIdx && ref.PlanId == newRef.PlanId {
			return int32(idx)
		}
	}

	idx := int32(len(s.QueryPlanReferences))
	s.QueryPlanReferences = append(s.QueryPlanReferences, &newRef)

	return idx
}

func transformPostgresPlans(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState, queryIDKeyToIdx QueryIDKeyToIdx) snapshot.FullSnapshot {
	groupedPlans := groupPlans(transientState.Plans, diffState.PlanStats, queryIDKeyToIdx)
	for key, value := range groupedPlans {
		idx := upsertQueryPlanReference(&s, key.queryIdx, key.planID)

		var planType snapshot.QueryPlanInformation_PlanType
		switch value.plan.PlanType {
		case "no plan":
			planType = snapshot.QueryPlanInformation_NO_PLAN
		case "estimate":
			planType = snapshot.QueryPlanInformation_ESTIMATE
		case "actual":
			planType = snapshot.QueryPlanInformation_ACTUAL
		}
		info := snapshot.QueryPlanInformation{
			QueryPlanIdx:     idx,
			ExplainPlan:      value.plan.ExplainPlan,
			PlanCapturedTime: timestamppb.New(value.plan.PlanCapturedTime),
			PlanType:         planType,
		}
		s.QueryPlanInformations = append(s.QueryPlanInformations, &info)

		// Plan stats (from a full snapshot run)
		stats := transformQueryPlanStatistic(value.planStats, idx)
		s.QueryPlanStatistics = append(s.QueryPlanStatistics, &stats)
	}

	// Historic plan stats (similar to historic statement stats, collected every 1 min)
	for timeKey, diffedStats := range transientState.HistoricPlanStats {
		// Ignore any data older than an hour, as a safety measure in case of many
		// failed full snapshot runs (which don't reset state)
		if time.Since(timeKey.CollectedAt).Hours() >= 1 {
			continue
		}

		h := snapshot.HistoricQueryPlanStatistics{}
		h.CollectedAt = timestamppb.New(timeKey.CollectedAt)
		h.CollectedIntervalSecs = timeKey.CollectedIntervalSecs

		groupedPlans := groupPlans(transientState.Plans, diffedStats, queryIDKeyToIdx)
		for key, value := range groupedPlans {
			idx := upsertQueryPlanReference(&s, key.queryIdx, key.planID)
			stats := transformQueryPlanStatistic(value.planStats, idx)
			h.Statistics = append(h.Statistics, &stats)
		}
		s.HistoricQueryPlanStatistics = append(s.HistoricQueryPlanStatistics, &h)
	}

	return s
}
