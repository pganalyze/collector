package runner

import (
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func diffState(logger *util.Logger, prevState state.PersistedState, newState state.PersistedState, collectedIntervalSecs uint32) (diffState state.DiffState) {
	diffState.StatementStats = diffStatements(newState.StatementStats, prevState.StatementStats)
	diffState.RelationStats = diffRelationStats(newState.RelationStats, prevState.RelationStats)
	diffState.IndexStats = diffIndexStats(newState.IndexStats, prevState.IndexStats)
	diffState.SystemCPUStats = diffSystemCPUStats(newState.System.CPUStats, prevState.System.CPUStats)
	diffState.SystemNetworkStats = diffSystemNetworkStats(newState.System.NetworkStats, prevState.System.NetworkStats, collectedIntervalSecs)
	diffState.SystemDiskStats = diffSystemDiskStats(newState.System.DiskStats, prevState.System.DiskStats, collectedIntervalSecs)
	diffState.CollectorStats = diffCollectorStats(newState.CollectorStats, prevState.CollectorStats)

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

func diffRelationStats(new state.PostgresRelationStatsMap, prev state.PostgresRelationStatsMap) (diff state.DiffedPostgresRelationStatsMap) {
	followUpRun := len(prev) > 0

	diff = make(state.DiffedPostgresRelationStatsMap)
	for key, stats := range new {
		prevStats, exists := prev[key]
		if exists {
			diff[key] = stats.DiffSince(prevStats)
		} else if followUpRun { // New since the last run
			diff[key] = stats.DiffSince(state.PostgresRelationStats{})
		} else {
			diff[key] = state.DiffedPostgresRelationStats{
				SizeBytes:        stats.SizeBytes,
				ToastSizeBytes:   stats.ToastSizeBytes,
				NLiveTup:         stats.NLiveTup,
				NDeadTup:         stats.NDeadTup,
				NModSinceAnalyze: stats.NModSinceAnalyze,
				LastVacuum:       stats.LastVacuum,
				LastAutovacuum:   stats.LastAutovacuum,
				LastAnalyze:      stats.LastAnalyze,
				LastAutoanalyze:  stats.LastAutoanalyze,
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
		if exists {
			diff[key] = stats.DiffSince(prevStats)
		} else if followUpRun { // New since the last run
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
