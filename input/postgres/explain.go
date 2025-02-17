package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func RunExplain(ctx context.Context, server *state.Server, inputs []state.PostgresQuerySample, collectionOpts state.CollectionOpts, logger *util.Logger) (outputs []state.PostgresQuerySample) {
	var samplesByDb = make(map[string]([]state.PostgresQuerySample))

	skip := func(sample state.PostgresQuerySample) bool {
		monitoredDb := sample.Database == "" || sample.Database == server.Config.GetDbName() ||
			server.Config.DbAllNames || contains(server.Config.DbExtraNames, sample.Database)

		return !monitoredDb ||
			// Ignore collector queries
			strings.HasPrefix(sample.Query, QueryMarkerSQL) ||
			// Ignore backup-related queries (they usually take long but not because of something that can be EXPLAINed)
			strings.Contains(sample.Query, "pg_start_backup") ||
			strings.Contains(sample.Query, "pg_stop_backup")
	}

	for _, sample := range inputs {
		if skip(sample) {
			continue
		}
		if sample.HasExplain { // EXPLAIN was already collected, e.g. from auto_explain
			outputs = append(outputs, sample)
			continue
		}
		samplesByDb[sample.Database] = append(samplesByDb[sample.Database], sample)
	}

	for dbName, dbSamples := range samplesByDb {
		dbOutputs := runExplainForDb(ctx, server, collectionOpts, logger, dbName, dbSamples)
		outputs = append(outputs, dbOutputs...)
	}
	return
}

func runExplainForDb(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, dbName string, dbSamples []state.PostgresQuerySample) (outputs []state.PostgresQuerySample) {
	db, err := EstablishConnection(ctx, server, logger, collectionOpts, dbName)
	if err != nil {
		logger.PrintVerbose("Could not connect to %s to run explain: %s; skipping", dbName, err)
		return nil
	}
	defer db.Close()

	c, err := NewCollection(ctx, logger, server, collectionOpts, db)
	if err != nil {
		logger.PrintError("Error setting up collection: %s", err)
		return nil
	}

	useHelper := StatsHelperExists(ctx, db, "explain")
	if useHelper {
		logger.PrintVerbose("Found pganalyze.explain() stats helper in database \"%s\"", dbName)
	}

	dbOutputs, err := runExplainForSample(ctx, db, dbSamples, useHelper)
	if err != nil {
		logger.PrintVerbose("Failed to run explain: %s", err)
		return nil
	}

	hasPermErr := false
	for _, sample := range dbOutputs {
		if strings.HasPrefix(sample.ExplainError, "pq: permission denied") {
			hasPermErr = true
			break
		}
	}
	if hasPermErr && c.ConnectedAsSuperUser {
		logger.PrintInfo("Warning: pganalyze.explain() helper function not found in database \"%s\". Please set up"+
			" the monitoring helper functions (https://pganalyze.com/docs/explain/setup/log_explain/01_create_helper_functions)"+
			" in every database you want to monitor to avoid permissions issues when running log-based EXPLAIN.", dbName)
	}

	return dbOutputs
}

func runExplainForSample(ctx context.Context, db *sql.DB, inputs []state.PostgresQuerySample, useHelper bool) ([]state.PostgresQuerySample, error) {
	var outputs []state.PostgresQuerySample
	for _, sample := range inputs {
		var isUtil []bool
		// To be on the safe side never EXPLAIN a statement that can't be parsed,
		// or multiple statements in one (leading to accidental execution)
		isUtil, err := util.IsUtilityStmt(sample.Query)
		if err == nil && len(isUtil) == 1 && !isUtil[0] {
			var explainOutput []byte

			sample.HasExplain = true
			sample.ExplainSource = pganalyze_collector.QuerySample_STATEMENT_LOG_EXPLAIN_SOURCE
			sample.ExplainFormat = pganalyze_collector.QuerySample_JSON_EXPLAIN_FORMAT

			if useHelper {
				err = db.QueryRowContext(ctx, QueryMarkerSQL+"SELECT pganalyze.explain($1, $2)", sample.Query, pq.Array(sample.Parameters)).Scan(&explainOutput)
				if err != nil {
					if ctx.Err() != nil {
						return nil, err
					}
					sample.ExplainError = fmt.Sprintf("%s", err)
				}
			} else {
				if len(sample.Parameters) > 0 {
					_, err = db.ExecContext(ctx, QueryMarkerSQL+"PREPARE pganalyze_explain AS "+sample.Query)
					if err != nil {
						if ctx.Err() != nil {
							return nil, err
						}
						sample.ExplainError = fmt.Sprintf("%s", err)
					} else {
						paramStr := getQuotedParamsStr(sample.Parameters)
						err = db.QueryRowContext(ctx, QueryMarkerSQL+"EXPLAIN (VERBOSE, FORMAT JSON) EXECUTE pganalyze_explain("+paramStr+")").Scan(&explainOutput)
						if err != nil {
							if ctx.Err() != nil {
								return nil, err
							}
							sample.ExplainError = fmt.Sprintf("%s", err)
						}

						_, err = db.ExecContext(ctx, QueryMarkerSQL+"DEALLOCATE pganalyze_explain")
						if err != nil {
							return nil, err
						}
					}
				} else {
					err = db.QueryRowContext(ctx, QueryMarkerSQL+"EXPLAIN (VERBOSE, FORMAT JSON) "+sample.Query).Scan(&explainOutput)
					if err != nil {
						if ctx.Err() != nil {
							return nil, err
						}
						sample.ExplainError = fmt.Sprintf("%s", err)
					}
				}
			}

			if len(explainOutput) > 0 {
				var explainOutputJSON []state.ExplainPlanContainer
				if err := json.Unmarshal(explainOutput, &explainOutputJSON); err != nil {
					sample.ExplainError = fmt.Sprintf("%s", err)
				} else if len(explainOutputJSON) != 1 {
					sample.ExplainError = fmt.Sprintf("Unexpected plan size: %d (expected 1)", len(explainOutputJSON))
				} else {
					sample.ExplainOutputJSON = &explainOutputJSON[0]
				}
			}
		}

		outputs = append(outputs, sample)
	}

	return outputs, nil
}

func contains(strs []string, val string) bool {
	for _, str := range strs {
		if str == val {
			return true
		}
	}
	return false
}

func getQuotedParamsStr(parameters []null.String) string {
	params := []string{}
	for i := 0; i < len(parameters); i++ {
		if parameters[i].Valid {
			params = append(params, pq.QuoteLiteral(parameters[i].String))
		} else {
			params = append(params, "NULL")
		}
	}
	return strings.Join(params, ", ")
}
