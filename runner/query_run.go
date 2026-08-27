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

func SetupQueryRunnerForAllServers(servers []*state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) {
	for idx := range servers {
		SetupQueryRunnerForServer(servers[idx], collectionOpts, logger)
	}
}

func SetupQueryRunnerForServer(server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) {
	if collectionOpts.ForceEmptyGrant {
		return
	}
	go func() {
		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)
		cleanupInterval := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <-server.Ctx.Done():
				return
			case <-cleanupInterval.C:
				cleanup(server)
			default:
				run(server.Ctx, server, collectionOpts, prefixedLogger)
				time.Sleep(1 * time.Second)
			}
		}
	}()
}

func run(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, query := range pendingQueryRuns(server) {
		server.QueryRunsMutex.Lock()
		query.StartedAt = time.Now()
		server.QueryRunsMutex.Unlock()
		logger.PrintVerbose("Query run %d starting: %s", query.Id, query.QueryText)

		result, err := runQueryOnDatabase(ctx, server, collectionOpts, logger, query)
		if err != nil {
			server.QueryRunsMutex.Lock()
			query.FinishedAt = time.Now()
			query.Error = err.Error()
			server.QueryRunsMutex.Unlock()
			continue
		}

		server.QueryRunsMutex.Lock()
		query.FinishedAt = time.Now()
		query.Result = result
		queryRun := *query
		server.QueryRunsMutex.Unlock()

		// Activity snapshots will eventually send the query run result, but to reduce latency
		// we also send a query run snapshot immediately after the query has finished.
		output.SubmitQueryRunSnapshot(ctx, server, collectionOpts, logger, queryRun)
	}
}

// Returns the query runs that have not been run yet
//
// The map must be read whilst holding the mutex, since the WebSocket message handler
// adds new query runs to it concurrently.
func pendingQueryRuns(server *state.Server) []*state.QueryRun {
	var pending []*state.QueryRun

	server.QueryRunsMutex.Lock()
	defer server.QueryRunsMutex.Unlock()

	for _, query := range server.QueryRuns {
		if query.FinishedAt.IsZero() {
			pending = append(pending, query)
		}
	}

	return pending
}

func runQueryOnDatabase(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, query *state.QueryRun) (string, error) {
	if query.Type != pganalyze_collector.QueryRunType_EXPLAIN {
		logger.PrintVerbose("Unhandled query run type %d for %d", query.Type, query.Id)
		return "", errors.New("Unhandled query run type")
	}

	db, err := postgres.EstablishConnection(ctx, server, logger, collectionOpts, query.DatabaseName)
	if err != nil {
		return "", err
	}
	defer db.Close()

	h, err := postgres.NewCollection(ctx, logger, server, collectionOpts, db)
	if err != nil {
		return "", err
	}

	if h.HelperExists("explain_analyze", []string{"text", "text[]", "text[]", "text[]"}) {
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
	query.BackendPid = pid
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
