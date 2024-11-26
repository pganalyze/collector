package runner

import (
	"context"
	"errors"
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
	for id, query := range server.QueryRuns {
		if !query.FinishedAt.IsZero() {
			continue
		}
		server.QueryRunsMutex.Lock()
		server.QueryRuns[id].StartedAt = time.Now()
		server.QueryRunsMutex.Unlock()
		logger.PrintVerbose("Query run %d starting: %s", query.Id, query.QueryText)

		db, err := postgres.EstablishConnection(ctx, server, logger, collectionOpts, query.DatabaseName)
		if err != nil {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[id].FinishedAt = time.Now()
			server.QueryRuns[id].Error = err.Error()
			server.QueryRunsMutex.Unlock()
			continue
		}
		defer db.Close()

		err = postgres.SetStatementTimeout(ctx, db, 60*1000)
		if err != nil {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[id].FinishedAt = time.Now()
			server.QueryRuns[id].Error = err.Error()
			server.QueryRunsMutex.Unlock()
			continue
		}

		pid := 0
		err = db.QueryRow(postgres.QueryMarkerSQL + "SELECT pg_backend_pid()").Scan(&pid)
		if err == nil {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[id].BackendPid = pid
			server.QueryRunsMutex.Unlock()
		} else {
			server.QueryRunsMutex.Lock()
			server.QueryRuns[id].FinishedAt = time.Now()
			server.QueryRuns[id].Error = err.Error()
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

		if query.Type == pganalyze_collector.QueryRunType_EXPLAIN {
			// Rollback any changes the query may perform
			db.ExecContext(ctx, postgres.QueryMarkerSQL+"BEGIN READ ONLY")

			err = db.QueryRowContext(ctx, comment+prefix+query.QueryText).Scan(&result)

			if err == nil {
				// Run EXPLAIN ANALYZE a second time to get a warm cache result
				err = db.QueryRowContext(ctx, comment+prefix+query.QueryText).Scan(&result)
			} else if strings.Contains(err.Error(), "statement timeout") {
				// If the EXPLAIN ANALYZE timed out, capture a regular EXPLAIN instead
				prefix = "EXPLAIN (VERBOSE, FORMAT JSON) "
				err = db.QueryRowContext(ctx, comment+prefix+query.QueryText).Scan(&result)
			}
		} else {
			err = errors.New("Unhandled query run type")
			logger.PrintVerbose("Unhandled query run type %d for %d", query.Type, query.Id)
		}

		server.QueryRunsMutex.Lock()
		server.QueryRuns[id].FinishedAt = time.Now()
		server.QueryRuns[id].Result = result
		if err != nil {
			server.QueryRuns[id].Error = err.Error()
		}
		server.QueryRunsMutex.Unlock()
	}
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
