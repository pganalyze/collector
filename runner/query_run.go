package runner

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
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
					run(ctx, server, collectionOpts, logger)
					time.Sleep(1 * time.Second)
				}
			}
		}(servers[idx])
	}
}

func run(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, query := range pendingQueryRuns(server) {
		runCtx, ok := startQueryRun(ctx, server, query)
		if !ok {
			continue
		}
		logger.PrintVerbose("Query run %d starting: %s", query.Id, query.QueryText)

		result, err := runQueryOnDatabase(runCtx, server, collectionOpts, logger, query)

		// Activity snapshots will eventually send the query run result, but to reduce latency
		// we also send a query run snapshot immediately after the query has finished.
		if queryRun, submit := finishQueryRun(server, query, result, err); submit {
			output.SubmitQueryRunSnapshot(ctx, server, collectionOpts, logger, queryRun)
		}
	}
}

// Returns the query runs that have not been run yet, in the order the server created them
// (IDs are assigned sequentially by the server)
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

	slices.SortFunc(pending, func(a, b *state.QueryRun) int {
		return cmp.Compare(a.Id, b.Id)
	})

	return pending
}

// Marks the query run as started and returns the context to run it with, unless it was
// canceled after pendingQueryRuns returned it. finishQueryRun releases the context.
func startQueryRun(ctx context.Context, server *state.Server, query *state.QueryRun) (context.Context, bool) {
	server.QueryRunsMutex.Lock()
	defer server.QueryRunsMutex.Unlock()

	if !query.FinishedAt.IsZero() {
		return nil, false
	}

	runCtx, cancel := context.WithCancel(ctx)
	query.StartedAt = time.Now()
	query.Cancel = cancel
	return runCtx, true
}

// Records the outcome of the query run, unless it was canceled while running, and returns a
// copy of the run along with whether to submit it right away
func finishQueryRun(server *state.Server, query *state.QueryRun, result string, err error) (state.QueryRun, bool) {
	server.QueryRunsMutex.Lock()
	defer server.QueryRunsMutex.Unlock()

	query.Cancel()
	query.Cancel = nil

	if !query.FinishedAt.IsZero() {
		// Canceled by the server (see cancelQueryRuns)
		return *query, false
	}

	query.FinishedAt = time.Now()
	if err != nil {
		query.Error = err.Error()
		return *query, false
	}
	query.Result = result
	return *query, true
}

// Stops the given query runs, whether or not they've started
func cancelQueryRuns(server *state.Server, ids []int64, logger *util.Logger) {
	server.QueryRunsMutex.Lock()
	defer server.QueryRunsMutex.Unlock()

	for _, id := range ids {
		query, exists := server.QueryRuns[id]
		if !exists || !query.FinishedAt.IsZero() {
			continue
		}
		logger.PrintVerbose("Query run %d canceled", id)
		query.FinishedAt = time.Now()
		query.Error = "Query run was canceled"
		if query.Cancel != nil {
			query.Cancel()
		}
	}
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
