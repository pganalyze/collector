package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	pg_query "github.com/lfittl/pg_query_go"
	pg_query_nodes "github.com/lfittl/pg_query_go/nodes"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func RunExplain(server state.Server, inputs []state.PostgresQuerySample, collectionOpts state.CollectionOpts, logger *util.Logger) (outputs []state.PostgresQuerySample) {
	var samplesByDb = make(map[string]([]state.PostgresQuerySample))

	for _, sample := range inputs {
		if sample.Database != server.Config.GetDbName() && !contains(server.Config.DbExtraNames, sample.Database) && !server.Config.DbAllNames {
			continue
		}
		samplesByDb[sample.Database] = append(samplesByDb[sample.Database], sample)
	}

	for dbName, dbSamples := range samplesByDb {
		db, err := EstablishConnection(server, logger, collectionOpts, dbName)

		if err == nil {
			dbOutputs := runDbExplain(db, dbSamples)
			outputs = append(outputs, dbOutputs...)

			db.Close()
		}
	}
	return
}

func runDbExplain(db *sql.DB, inputs []state.PostgresQuerySample) (outputs []state.PostgresQuerySample) {
	for _, sample := range inputs {
		// EXPLAIN was already collected, e.g. from auto_explain
		if sample.HasExplain {
			continue
		}

		// Ignore collector queries
		if strings.HasPrefix(sample.Query, QueryMarkerSQL) {
			continue
		}

		// Ignore backup-related queries (they usually take long but not because of something that can be EXPLAINed)
		if strings.Contains(sample.Query, "pg_start_backup") || strings.Contains(sample.Query, "pg_stop_backup") {
			continue
		}

		// To be on the safe side never EXPLAIN a statement that can't be parsed,
		// or multiple statements in one (leading to accidental execution)
		parsetree, err := pg_query.Parse(sample.Query)
		if err == nil && len(parsetree.Statements) == 1 {
			stmt := parsetree.Statements[0].(pg_query_nodes.RawStmt).Stmt
			switch stmt.(type) {
			case pg_query_nodes.SelectStmt, pg_query_nodes.InsertStmt, pg_query_nodes.UpdateStmt, pg_query_nodes.DeleteStmt:
				sample.HasExplain = true
				sample.ExplainSource = pganalyze_collector.QuerySample_STATEMENT_LOG_EXPLAIN_SOURCE
				sample.ExplainFormat = pganalyze_collector.QuerySample_JSON_EXPLAIN_FORMAT

				if len(sample.Parameters) > 0 {
					_, err = db.Exec(QueryMarkerSQL + "PREPARE pganalyze_explain AS " + sample.Query)
					if err != nil {
						sample.ExplainError = fmt.Sprintf("%s", err)
						continue
					}

					params := []string{}
					for i := 0; i < len(sample.Parameters); i++ {
						params = append(params, pq.QuoteLiteral(sample.Parameters[i]))
					}
					err = db.QueryRow(QueryMarkerSQL + "EXPLAIN (VERBOSE, FORMAT JSON) EXECUTE pganalyze_explain(" + strings.Join(params, ", ") + ")").Scan(&sample.ExplainOutput)
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
