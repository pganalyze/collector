package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SetupQueryRunnerForAllServers(ctx context.Context, servers []*state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) {
	if collectionOpts.ForceEmptyGrant {
		return
	}
	for idx := range servers {
		go func(server *state.Server) {
			logger = logger.WithPrefixAndRememberErrors(server.Config.SectionName)
			cleanupInterval := time.NewTicker(5 * time.Minute)
			for {
				select {
				case <-ctx.Done():
					return
				case <-cleanupInterval.C:
					cleanup(server)
				default:
					if server.Config.EnableQueryRunner {
						run(ctx, server, collectionOpts, logger)
					}
					time.Sleep(1 * time.Second)
				}
			}
		}(servers[idx])
	}
}

func run(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) {
	for id, query := range server.QueryRuns {
		if !query.FinishedAt.IsZero() {
			continue
		}

		server.QueryRunsMutex.Lock()
		server.QueryRuns[id].StartedAt = time.Now()
		server.QueryRunsMutex.Unlock()
		logger.PrintVerbose("Query run %d starting: %s", query.Id, query.QueryText)

		result, err := runQueryOnDatabase(ctx, server, collectionOpts, logger, id, query)
		if err != nil {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[id].FinishedAt = time.Now()
			server.QueryRuns[id].Error = err.Error()
			server.QueryRunsMutex.Unlock()
			continue
		}

		server.QueryRunsMutex.Lock()
		server.QueryRuns[id].FinishedAt = time.Now()
		server.QueryRuns[id].Result = result
		server.QueryRunsMutex.Unlock()

		// Activity snapshots will eventually send the query run result, but to reduce latency
		// we also send a query run snapshot immediately after the query has finished.
		output.SubmitQueryRunSnapshot(ctx, server, collectionOpts, logger, *server.QueryRuns[id])
	}
}

func runQueryOnDatabase(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, id int64, query *state.QueryRun) (string, error) {
	if query.Type != pganalyze_collector.QueryRunType_EXPLAIN {
		logger.PrintVerbose("Unhandled query run type %d for %d", query.Type, query.Id)
		return "", errors.New("Unhandled query run type")
	}

	db, err := postgres.EstablishConnection(ctx, server, logger, collectionOpts, query.DatabaseName)
	if err != nil {
		return "", err
	}
	defer db.Close()

	if postgres.StatsHelperExists(ctx, db, "explain_analyze") {
		logger.PrintVerbose("Found pganalyze.explain_analyze helper function in database \"%s\"", query.DatabaseName)
	} else {
		return "", fmt.Errorf("Required helper function pganalyze.explain_analyze is not set up")
	}

	pid := 0
	err = db.QueryRow(postgres.QueryMarkerSQL + "SELECT pg_backend_pid()").Scan(&pid)
	if err != nil {
		return "", err
	}
	server.QueryRunsMutex.Lock()
	server.QueryRuns[id].BackendPid = pid
	server.QueryRunsMutex.Unlock()

	for name, value := range query.PostgresSettings {
		_, err = db.ExecContext(ctx, postgres.QueryMarkerSQL+fmt.Sprintf("SET %s = %s", pq.QuoteIdentifier(name), pq.QuoteLiteral(value)))
		if err != nil {
			return "", err
		}
	}

	err = postgres.SetStatementTimeout(ctx, db, 60*1000)
	if err != nil {
		return "", err
	}

	// We don't include QueryMarkerSQL so query runs are reported separately in pganalyze
	marker := fmt.Sprintf("/* pganalyze:no-alert,pganalyze-query-run:%d */ ", query.Id)

	return postgres.RunExplainAnalyzeForQueryRun(ctx, db, query.QueryText, query.QueryParameters, query.QueryParameterTypes, marker)
}

// Removes old query runs that have finished
func cleanup(server *state.Server) {
	server.QueryRunsMutex.Lock()
	queryRuns := make(map[int64]*state.QueryRun)
	for id, query := range server.QueryRuns {
		if query.FinishedAt.IsZero() || time.Since(query.FinishedAt) < 10*time.Minute {
			queryRuns[id] = query
		}
	}
	server.QueryRuns = queryRuns
	server.QueryRunsMutex.Unlock()
}
