package runner

import (
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func diffState(logger *util.Logger, prevState state.State, newState state.State) (diffState state.DiffState) {
	diffState.Statements = diffStatements(newState.Statements, prevState.Statements)
	diffState.RelationStats = diffRelationStats(newState.RelationStats, prevState.RelationStats)
	diffState.IndexStats = diffIndexStats(newState.IndexStats, prevState.IndexStats)

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
