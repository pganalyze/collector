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
		// TODO wait group?
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
	for id, query := range server.QueryRuns {
		if server.QueryRunActive == id && query.Canceled {
			// Canceling an already in progress query
			server.QueryRunsMutex.Lock()
			server.QueryRuns[id].FinishedAt = time.Now()
			server.QueryRuns[id].Error = "Canceled by user request"
			server.QueryRunActive = 0
			server.QueryRunCancel()
			server.QueryRunsMutex.Unlock()
		}
		if !query.FinishedAt.IsZero() || server.QueryRunActive != 0 {
			continue
		}
		server.QueryRunsMutex.Lock()
		server.QueryRuns[id].StartedAt = time.Now()
		if query.Canceled {
			// Query was canceled before it had a chance to start
			server.QueryRuns[id].FinishedAt = time.Now()
			server.QueryRuns[id].Error = "Canceled by user request"
			server.QueryRunsMutex.Unlock()
			continue
		}
		server.QueryRunActive = id
		ctx, cancelQuery := context.WithCancel(context.Background())
		server.QueryRunCancel = cancelQuery
		server.QueryRunsMutex.Unlock()
		logger.PrintVerbose("Query run %d starting: %s", query.Id, query.QueryText)

		go func() {
			result, err := runQueryOnDatabase(ctx, server, collectionOpts, logger, id, query)
			server.QueryRunsMutex.Lock()
			server.QueryRuns[id].FinishedAt = time.Now()
			server.QueryRuns[id].Result = result
			if err != nil {
				server.QueryRuns[id].Error = err.Error()
			}
			server.QueryRunActive = 0
			server.QueryRunCancel()
			server.QueryRunsMutex.Unlock()
			// Immediately send result to reduce latency
			output.SubmitQueryRunSnapshot(ctx, server, collectionOpts, logger, *server.QueryRuns[id])
		}()
	}
}

func runQueryOnDatabase(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, id int64, query *state.QueryRun) (string, error) {
	_, isValid := pganalyze_collector.QueryRunType_name[int32(query.Type)]
	if !isValid {
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

	// Type-specific validations
	if query.Type == pganalyze_collector.QueryRunType_EXPLAIN {
		if h.HelperExists("explain_analyze", []string{"text", "text[]", "text[]", "text[]"}) {
			logger.PrintVerbose("Found pganalyze.explain_analyze helper function in database \"%s\"", query.DatabaseName)
		} else {
			return "", fmt.Errorf("Required helper function pganalyze.explain_analyze is not set up")
		}
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

	if query.Type == pganalyze_collector.QueryRunType_EXPLAIN {
		return postgres.RunExplainAnalyzeForQueryRun(ctx, db, query.QueryText, query.QueryParameters, query.QueryParameterTypes, marker)
	} else if query.Type == pganalyze_collector.QueryRunType_PGSTATTUPLE {
		return postgres.RunPgstattupleForQueryRun(ctx, db, query.QueryText, marker)
	} else if query.Type == pganalyze_collector.QueryRunType_VACUUM {
		return postgres.RunVacuumForQueryRun(ctx, db, query.QueryText, marker)
	} else if query.Type == pganalyze_collector.QueryRunType_REINDEX {
		return postgres.RunReindexForQueryRun(ctx, db, query.QueryText, marker)
	} else if query.Type == pganalyze_collector.QueryRunType_REPACK {
		return postgres.RunRepackForQueryRun(ctx, db, query.QueryText, marker)
	} else {
		logger.PrintError("Unhandled query run type %d for %d", query.Type, query.Id)
		return "", errors.New("Unhandled query run type")
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
