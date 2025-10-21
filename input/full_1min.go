package input

import (
	"context"
	"database/sql"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pkg/errors"
)

// CollectAndDiff1minStats - Collects once-a-minute data of certain stats
func CollectAndDiff1minStats(ctx context.Context, c *postgres.Collection, connection *sql.DB, collectedAt time.Time, prevState state.PersistedHighFreqState) (state.PersistedHighFreqState, error) {
	var err error

	newState := prevState
	newState.LastStatementStatsAt = time.Now()

	_, _, newState.StatementStats, err = postgres.GetStatements(ctx, c, connection, false)
	if err != nil {
		return newState, errors.Wrap(err, "error collecting pg_stat_statements")
	}
	_, newState.PlanStats, err = postgres.GetPlans(ctx, c, connection, false)
	if err != nil {
		return newState, errors.Wrap(err, "error collecting query plan stats")
	}

	newState.ServerIoStats, err = postgres.GetPgStatIo(ctx, c, connection)
	if err != nil {
		return newState, errors.Wrap(err, "error collecting Postgres server statistics")
	}

	// Don't calculate any diffs on the first run (but still update the state)
	if len(prevState.StatementStats) == 0 || prevState.LastStatementStatsAt.IsZero() {
		return newState, nil
	}

	timeKey := state.HistoricStatsTimeKey{
		CollectedAt:           collectedAt,
		CollectedIntervalSecs: uint32(newState.LastStatementStatsAt.Sub(prevState.LastStatementStatsAt) / time.Second),
	}

	newState.UnidentifiedStatementStats = prevState.UnidentifiedStatementStats
	if newState.UnidentifiedStatementStats == nil {
		newState.UnidentifiedStatementStats = make(state.HistoricStatementStatsMap)
	}
	newState.UnidentifiedStatementStats[timeKey] = diffStatements(newState.StatementStats, prevState.StatementStats)

	newState.UnidentifiedPlanStats = prevState.UnidentifiedPlanStats
	if newState.UnidentifiedPlanStats == nil {
		newState.UnidentifiedPlanStats = make(state.HistoricPlanStatsMap)
	}
	newState.UnidentifiedPlanStats[timeKey] = diffPlanStats(newState.PlanStats, prevState.PlanStats)

	if c.PostgresVersion.Numeric >= state.PostgresVersion16 {
		newState.QueuedServerIoStats = prevState.QueuedServerIoStats
		if newState.QueuedServerIoStats == nil {
			newState.QueuedServerIoStats = make(state.HistoricPostgresServerIoStatsMap)
		}
		newState.QueuedServerIoStats[timeKey] = diffServerIoStats(newState.ServerIoStats, prevState.ServerIoStats)
	}

	return newState, nil
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
