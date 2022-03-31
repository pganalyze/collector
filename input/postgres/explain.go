package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func RunExplain(server *state.Server, inputs []state.PostgresQuerySample, collectionOpts state.CollectionOpts, logger *util.Logger) (outputs []state.PostgresQuerySample) {
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
		db, err := EstablishConnection(server, logger, collectionOpts, dbName)

		if err != nil {
			logger.PrintVerbose("Could not connect to %s to run explain: %s; skipping", dbName, err)
			continue
		}
		useHelper := StatsHelperExists(db, "explain")
		if useHelper {
			logger.PrintVerbose("Found pganalyze.explain() stats helper in database \"%s\"", dbName)
		}

		dbOutputs := runDbExplain(db, dbSamples, useHelper)
		db.Close()

		hasPermErr := false
		for _, sample := range dbOutputs {
			if strings.HasPrefix(sample.ExplainError, "pq: permission denied") {
				hasPermErr = true
				break
			}
		}
		if hasPermErr && !connectedAsSuperUser(db, server.Config.SystemType) {
			logger.PrintInfo("Warning: pganalyze.explain() helper function not found in database \"%s\". Please set up"+
				" the monitoring helper functions (https://github.com/pganalyze/collector#setting-up-a-restricted-monitoring-user)"+
				" in every database you want to monitor to avoid permissions issues when running log-based EXPLAIN.", dbName)
		}

		outputs = append(outputs, dbOutputs...)
	}
	return
}

func runDbExplain(db *sql.DB, inputs []state.PostgresQuerySample, useHelper bool) (outputs []state.PostgresQuerySample) {
	for _, sample := range inputs {
		// To be on the safe side never EXPLAIN a statement that can't be parsed,
		// or multiple statements in one (leading to accidental execution)
		isUtil, err := util.IsUtilityStmt(sample.Query)
		if err == nil && len(isUtil) == 1 && !isUtil[0] {
			sample.HasExplain = true
			sample.ExplainSource = pganalyze_collector.QuerySample_STATEMENT_LOG_EXPLAIN_SOURCE
			sample.ExplainFormat = pganalyze_collector.QuerySample_JSON_EXPLAIN_FORMAT

			if useHelper {
				err = db.QueryRow(QueryMarkerSQL+"SELECT pganalyze.explain($1, $2)", sample.Query, pq.Array(sample.Parameters)).Scan(&sample.ExplainOutput)
				if err != nil {
					sample.ExplainError = fmt.Sprintf("%s", err)
				}
			} else {
				if len(sample.Parameters) > 0 {
					_, err = db.Exec(QueryMarkerSQL + "PREPARE pganalyze_explain AS " + sample.Query)
					if err != nil {
						sample.ExplainError = fmt.Sprintf("%s", err)
						continue
					}

					paramStr := getQuotedParamsStr(sample.Parameters)
					err = db.QueryRow(QueryMarkerSQL + "EXPLAIN (VERBOSE, FORMAT JSON) EXECUTE pganalyze_explain(" + paramStr + ")").Scan(&sample.ExplainOutput)
					if err != nil {
						sample.ExplainError = fmt.Sprintf("%s", err)
					}

					db.Exec(QueryMarkerSQL + "DEALLOCATE pganalyze_explain")
				} else {
					err = db.QueryRow(QueryMarkerSQL + "EXPLAIN (VERBOSE, FORMAT JSON) " + sample.Query).Scan(&sample.ExplainOutput)
					if err != nil {
						sample.ExplainError = fmt.Sprintf("%s", err)
					}
				}
			}
		}

		outputs = append(outputs, sample)
	}

	return
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
