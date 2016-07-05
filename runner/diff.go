package runner

import (
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func diffState(logger *util.Logger, prevState state.State, newState state.State, collectedIntervalSecs uint32) (diffState state.DiffState) {
	diffState.Statements = diffStatements(newState.Statements, prevState.Statements)
	diffState.RelationStats = diffRelationStats(newState.RelationStats, prevState.RelationStats)
	diffState.IndexStats = diffIndexStats(newState.IndexStats, prevState.IndexStats)
	diffState.SystemCPUStats = diffSystemCPUStats(newState.System.CPUStats, prevState.System.CPUStats)
	diffState.SystemNetworkStats = diffSystemNetworkStats(newState.System.NetworkStats, prevState.System.NetworkStats)
	diffState.SystemDiskStats = diffSystemDiskStats(newState.System.DiskStats, prevState.System.DiskStats, collectedIntervalSecs)

	return
}

func diffStatements(new state.PostgresStatementMap, prev state.PostgresStatementMap) (diff []state.DiffedPostgresStatement) {
	followUpRun := len(prev) > 0

	for key, statement := range new {
		var diffedStatement state.DiffedPostgresStatement

		prevStatement, exists := prev[key]
		if exists {
			diffedStatement = statement.DiffSince(prevStatement)
		} else if followUpRun { // New statement since the last run
			diffedStatement = statement.DiffSince(state.PostgresStatement{})
		}

		if diffedStatement.Calls > 0 {
			diff = append(diff, diffedStatement)
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

func diffSystemNetworkStats(new state.NetworkStatsMap, prev state.NetworkStatsMap) (diff state.DiffedNetworkStatsMap) {
	diff = make(state.DiffedNetworkStatsMap)
	for interfaceName, stats := range new {
		prevStats, exists := prev[interfaceName]
		if exists {
			diff[interfaceName] = stats.DiffSince(prevStats)
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
