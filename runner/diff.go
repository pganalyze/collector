package runner

import (
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func diffState(logger *util.Logger, prevState state.PersistedState, newState state.PersistedState, collectedIntervalSecs uint32) (diffState state.DiffState) {
	diffState.StatementStats = diffStatements(newState.StatementStats, prevState.StatementStats)
	diffState.PlanStats = diffPlanStats(newState.PlanStats, prevState.PlanStats)
	diffState.SchemaStats = make(map[state.Oid]*state.DiffedSchemaStats)
	for dbOid := range newState.SchemaStats {
		newDbStats := newState.SchemaStats[dbOid]
		prevDbStats := prevState.SchemaStats[dbOid]
		var prevRelStats state.PostgresRelationStatsMap
		var prevIdxStats state.PostgresIndexStatsMap
		if prevDbStats != nil {
			prevRelStats = prevDbStats.RelationStats
			prevIdxStats = prevDbStats.IndexStats
		} else {
			prevRelStats = make(state.PostgresRelationStatsMap)
			prevIdxStats = make(state.PostgresIndexStatsMap)
		}
		diffState.SchemaStats[dbOid] = &state.DiffedSchemaStats{
			RelationStats: diffRelationStats(newDbStats.RelationStats, prevRelStats),
			IndexStats:    diffIndexStats(newDbStats.IndexStats, prevIdxStats),
		}
	}
	diffState.ServerIoStats = diffServerIoStats(newState.ServerIoStats, prevState.ServerIoStats)
	diffState.SystemCPUStats = diffSystemCPUStats(newState.System.CPUStats, prevState.System.CPUStats)
	diffState.SystemNetworkStats = diffSystemNetworkStats(newState.System.NetworkStats, prevState.System.NetworkStats, collectedIntervalSecs)
	diffState.SystemDiskStats = diffSystemDiskStats(newState.System.DiskStats, prevState.System.DiskStats, collectedIntervalSecs)
	diffState.CollectorStats = diffCollectorStats(newState.CollectorStats, prevState.CollectorStats)

	diffState.DatabaseStats = diffDatabaseStats(newState.DatabaseStats, prevState.DatabaseStats)
	diffState.PgStatStatementsStats = diffPgStatStatementsStats(newState.PgStatStatementsStats, prevState.PgStatStatementsStats)

	return
}

func diffStatements(new state.PostgresStatementStatsMap, prev state.PostgresStatementStatsMap) (diff state.DiffedPostgresStatementStatsMap) {
	followUpRun := len(prev) > 0
	diff = make(state.DiffedPostgresStatementStatsMap)

	for key, statement := range new {
		var diffedStatement state.DiffedPostgresStatementStats

		prevStatement, exists := prev[key]
		if exists {
			diffedStatement = statement.DiffSince(prevStatement)
		} else if followUpRun { // New statement since the last run
			diffedStatement = statement.DiffSince(state.PostgresStatementStats{})
		}

		if diffedStatement.Calls > 0 {
			diff[key] = diffedStatement
		}
	}

	return
}

func diffPlanStats(new state.PostgresPlanStatsMap, prev state.PostgresPlanStatsMap) (diff state.DiffedPostgresPlanStatsMap) {
	followUpRun := len(prev) > 0
	diff = make(state.DiffedPostgresPlanStatsMap)

	for key, planStats := range new {
		var diffedPlanStats state.DiffedPostgresStatementStats

		prevPlanStats, exists := prev[key]
		if exists {
			diffedPlanStats = planStats.DiffSince(prevPlanStats)
		} else if followUpRun { // New plan since the last run
			diffedPlanStats = planStats.DiffSince(state.PostgresStatementStats{})
		}

		if diffedPlanStats.Calls > 0 {
			diff[key] = diffedPlanStats
		}
	}

	return
}

func diffRelationStats(new state.PostgresRelationStatsMap, prev state.PostgresRelationStatsMap) (diff state.DiffedPostgresRelationStatsMap) {
	followUpRun := len(prev) > 0

	diff = make(state.DiffedPostgresRelationStatsMap)
	for key, stats := range new {
		prevStats, exists := prev[key]
		if stats.ExclusivelyLocked {
			// Skip, we don't have any usable data for this relation
		} else if exists && !prevStats.ExclusivelyLocked {
			diff[key] = stats.DiffSince(prevStats)
		} else if followUpRun && !prevStats.ExclusivelyLocked { // New relation since the last run
			diff[key] = stats.DiffSince(state.PostgresRelationStats{})
		} else {
			diff[key] = state.DiffedPostgresRelationStats{
				SizeBytes:        stats.SizeBytes,
				ToastSizeBytes:   stats.ToastSizeBytes,
				NLiveTup:         stats.NLiveTup,
				NDeadTup:         stats.NDeadTup,
				NModSinceAnalyze: stats.NModSinceAnalyze,
				NInsSinceVacuum:  stats.NInsSinceVacuum,
				LastVacuum:       stats.LastVacuum,
				LastAutovacuum:   stats.LastAutovacuum,
				LastAnalyze:      stats.LastAnalyze,
				LastAutoanalyze:  stats.LastAutoanalyze,
				FrozenXIDAge:     stats.FrozenXIDAge,
				MinMXIDAge:       stats.MinMXIDAge,
				Relpages:         stats.Relpages,
				Reltuples:        stats.Reltuples,
				Relallvisible:    stats.Relallvisible,
				ToastReltuples:   stats.ToastReltuples,
				ToastRelpages:    stats.ToastRelpages,
			}
		}
	}

	return
}

func diffIndexStats(new state.PostgresIndexStatsMap, prev state.PostgresIndexStatsMap) (diff state.DiffedPostgresIndexStatsMap) {
	followUpRun := len(prev) > 0

	diff = make(state.DiffedPostgresIndexStatsMap)
	for key, stats := range new {
		prevStats, exists := prev[key]
		if stats.ExclusivelyLocked {
			// Skip, we don't have any usable data for this index
		} else if exists && !prevStats.ExclusivelyLocked {
			diff[key] = stats.DiffSince(prevStats)
		} else if followUpRun && !prevStats.ExclusivelyLocked { // New index since the last run
			diff[key] = stats.DiffSince(state.PostgresIndexStats{})
		} else {
			diff[key] = state.DiffedPostgresIndexStats{
				SizeBytes: stats.SizeBytes,
			}
		}
	}

	return
}

func diffSystemCPUStats(new state.CPUStatisticMap, prev state.CPUStatisticMap) (diff state.DiffedSystemCPUStatsMap) {
	diff = make(state.DiffedSystemCPUStatsMap)
	for cpuID, stats := range new {
		if stats.DiffedOnInput {
			if stats.DiffedValues != nil {
				diff[cpuID] = *stats.DiffedValues
			}
		} else {
			prevStats, exists := prev[cpuID]
			if exists {
				diff[cpuID] = stats.DiffSince(prevStats)
			}
		}
	}

	return
}

func diffSystemNetworkStats(new state.NetworkStatsMap, prev state.NetworkStatsMap, collectedIntervalSecs uint32) (diff state.DiffedNetworkStatsMap) {
	diff = make(state.DiffedNetworkStatsMap)
	for interfaceName, stats := range new {
		if stats.DiffedOnInput {
			if stats.DiffedValues != nil {
				diff[interfaceName] = *stats.DiffedValues
			}
		} else {
			prevStats, exists := prev[interfaceName]
			if exists {
				diff[interfaceName] = stats.DiffSince(prevStats, collectedIntervalSecs)
			}
		}
	}

	return
}

func diffSystemDiskStats(new state.DiskStatsMap, prev state.DiskStatsMap, collectedIntervalSecs uint32) (diff state.DiffedDiskStatsMap) {
	diff = make(state.DiffedDiskStatsMap)
	for deviceName, stats := range new {
		if stats.DiffedOnInput {
			if stats.DiffedValues != nil {
				diff[deviceName] = *stats.DiffedValues
			}
		} else {
			prevStats, exists := prev[deviceName]
			if exists {
				diff[deviceName] = stats.DiffSince(prevStats, collectedIntervalSecs)
			}
		}
	}

	return
}

func diffCollectorStats(new state.CollectorStats, prev state.CollectorStats) (diff state.DiffedCollectorStats) {
	diff = new.DiffSince(prev)
	return
}

func diffPgStatStatementsStats(new state.PgStatStatementsStats, prev state.PgStatStatementsStats) (diff state.DiffedPgStatStatementsStats) {
	diff = new.DiffSince(prev)
	return
}

func diffDatabaseStats(new state.PostgresDatabaseStatsMap, prev state.PostgresDatabaseStatsMap) (diff state.DiffedPostgresDatabaseStatsMap) {
	followUpRun := len(prev) > 0

	diff = make(state.DiffedPostgresDatabaseStatsMap)
	for databaseOid, stats := range new {
		prevStats, exists := prev[databaseOid]
		if exists {
			diff[databaseOid] = stats.DiffSince(prevStats)
		} else if followUpRun { // New since the last run
			diff[databaseOid] = stats.DiffSince(state.PostgresDatabaseStats{})
		} else {
			diff[databaseOid] = state.DiffedPostgresDatabaseStats{
				FrozenXIDAge: stats.FrozenXIDAge,
				MinMXIDAge:   stats.MinMXIDAge,
			}
		}
	}
	return
}

func diffServerIoStats(new state.PostgresServerIoStatsMap, prev state.PostgresServerIoStatsMap) (diff state.DiffedPostgresServerIoStatsMap) {
	followUpRun := len(prev) > 0

	diff = make(state.DiffedPostgresServerIoStatsMap)
	for k, stats := range new {
		var s state.DiffedPostgresServerIoStats
		prevStats, exists := prev[k]
		if exists {
			s = stats.DiffSince(prevStats)
		} else if followUpRun { // New since the last run
			s = stats.DiffSince(state.PostgresServerIoStats{})
		}
		// Skip over empty diffs (which can occur either because there was no activity, or for fixed entries that never saw activity)
		if s.Reads != 0 || s.Writes != 0 || s.Writebacks != 0 || s.Extends != 0 ||
			s.Hits != 0 || s.Evictions != 0 || s.Reuses != 0 || s.Fsyncs != 0 {
			diff[k] = s
		}
	}

	return
}
