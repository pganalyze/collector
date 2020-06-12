package runner

import (
	"database/sql"
	"sync"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pkg/errors"
)

func gatherQueryStatsForServer(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (state.PersistedState, error) {
	var err error
	var connection *sql.DB

	newState := server.PrevState
	systemType := server.Config.SystemType
	collectedAt := time.Now()

	connection, err = postgres.EstablishConnection(server, logger, globalCollectionOpts, "")
	if err != nil {
		return newState, errors.Wrap(err, "failed to connect to database")
	}

	defer connection.Close()

	postgresVersion, err := postgres.GetPostgresVersion(logger, connection)
	if err != nil {
		return newState, errors.Wrap(err, "error collecting Postgres Version")
	}

	if postgresVersion.Numeric < state.PostgresVersion94 {
		logger.PrintVerbose("Skipping high frequency query stats run since Postgres version is too old (9.4+ required)")
		return newState, nil
	}

	newState.LastStatementStatsAt = time.Now()
	_, _, newState.StatementStats, err = postgres.GetStatements(logger, connection, globalCollectionOpts, postgresVersion, false, systemType)
	if err != nil {
		return newState, errors.Wrap(err, "error collecting pg_stat_statements")
	}

	// Don't calculate any diffs on the first run (but still update the state)
	if len(server.PrevState.StatementStats) == 0 || server.PrevState.LastStatementStatsAt.IsZero() {
		return newState, nil
	}

	diffedStatementStats := diffStatements(newState.StatementStats, server.PrevState.StatementStats)
	collectedIntervalSecs := uint32(newState.LastStatementStatsAt.Sub(server.PrevState.LastStatementStatsAt) / time.Second)

	timeKey := state.PostgresStatementStatsTimeKey{CollectedAt: collectedAt, CollectedIntervalSecs: collectedIntervalSecs}
	newState.UnidentifiedStatementStats = server.PrevState.UnidentifiedStatementStats
	if newState.UnidentifiedStatementStats == nil {
		newState.UnidentifiedStatementStats = make(state.HistoricStatementStatsMap)
	}
	newState.UnidentifiedStatementStats[timeKey] = diffedStatementStats

	return newState, nil
}

func GatherQueryStatsFromAllServers(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	var wg sync.WaitGroup

	for idx := range servers {
		if servers[idx].Config.QueryStatsInterval != 60 {
			continue
		}

		wg.Add(1)
		go func(server *state.Server) {
			prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)

			server.StateMutex.Lock()
			newState, err := gatherQueryStatsForServer(*server, globalCollectionOpts, prefixedLogger)

			if err != nil {
				server.StateMutex.Unlock()
				prefixedLogger.PrintError("Could not collect query stats for server: %s", err)
				if server.Config.ErrorCallback != "" {
					go runCompletionCallback("error", server.Config.ErrorCallback, server.Config.SectionName, "query_stats", err, prefixedLogger)
				}
			} else {
				server.PrevState = newState
				server.StateMutex.Unlock()
				prefixedLogger.PrintVerbose("Successfully collected high frequency query statistics")
				if server.Config.SuccessCallback != "" {
					go runCompletionCallback("success", server.Config.SuccessCallback, server.Config.SectionName, "query_stats", nil, prefixedLogger)
				}
			}
			wg.Done()
		}(&servers[idx])
	}

	wg.Wait()

	return
}
