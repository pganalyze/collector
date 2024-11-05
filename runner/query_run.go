package runner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pganalyze/collector/input/postgres"
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
	for idx, query := range server.QueryRuns {
		if !query.FinishedAt.IsZero() {
			continue
		}
		server.QueryRunsMutex.Lock()
		server.QueryRuns[idx].StartedAt = time.Now()
		server.QueryRunsMutex.Unlock()
		logger.PrintVerbose("Query run %d starting: %s", query.Id, query.QueryText)

		db, err := postgres.EstablishConnection(ctx, server, logger, collectionOpts, query.DatabaseName)
		if err != nil {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[idx].FinishedAt = time.Now()
			server.QueryRuns[idx].Error = err.Error()
			server.QueryRunsMutex.Unlock()
			continue
		}
		defer db.Close()

		err = postgres.SetStatementTimeout(ctx, db, 60*1000)
		if err != nil {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[idx].FinishedAt = time.Now()
			server.QueryRuns[idx].Error = err.Error()
			server.QueryRunsMutex.Unlock()
			continue
		}

		pid := 0
		err = db.QueryRow(postgres.QueryMarkerSQL + "SELECT pg_backend_pid()").Scan(&pid)
		if err == nil {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[idx].BackendPid = pid
			server.QueryRunsMutex.Unlock()
		} else {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[idx].FinishedAt = time.Now()
			server.QueryRuns[idx].Error = err.Error()
			server.QueryRunsMutex.Unlock()
			continue
		}

		// We don't include QueryMarkerSQL so query runs are reported separately in pganalyze
		comment := fmt.Sprintf("/* pganalyze:no-alert,pganalyze-query-run:%d */ ", query.Id)
		prefix := ""
		result := ""
		if query.Type == pganalyze_collector.QueryRunType_EXPLAIN {
			prefix = "EXPLAIN (ANALYZE, VERBOSE, BUFFERS, FORMAT JSON) "
		}

		err = db.QueryRowContext(ctx, comment+prefix+query.QueryText).Scan(&result)

		if query.Type == pganalyze_collector.QueryRunType_EXPLAIN {
			// Run EXPLAIN ANALYZE a second time to get a warm cache result
			err = db.QueryRowContext(ctx, comment+prefix+query.QueryText).Scan(&result)

			// If the EXPLAIN ANALYZE timed out, capture a regular EXPLAIN instead
			if err != nil && strings.Contains(err.Error(), "statement timeout") {
				prefix = "EXPLAIN (VERBOSE, FORMAT JSON) "
				err = db.QueryRowContext(ctx, comment+prefix+query.QueryText).Scan(&result)
			}
		}

		server.QueryRunsMutex.Lock()
		server.QueryRuns[idx].FinishedAt = time.Now()
		server.QueryRuns[idx].Result = result
		if err != nil {
			server.QueryRuns[idx].Error = err.Error()
		}
		server.QueryRunsMutex.Unlock()
	}
}

// Removes old query runs that have finished
func cleanup(server *state.Server) {
	server.QueryRunsMutex.Lock()
	queryRuns := make([]state.QueryRun, 0)
	for _, query := range server.QueryRuns {
		if time.Since(query.FinishedAt) >= 10*time.Minute {
			queryRuns = append(queryRuns, query)
		}
	}
	server.QueryRuns = queryRuns
	server.QueryRunsMutex.Unlock()
}
