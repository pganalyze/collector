package postgres

import (
	"database/sql"
	"fmt"

	pg_query "github.com/lfittl/pg_query_go"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func RunExplain(db *sql.DB, inputs []state.PostgresQuerySample) (outputs []state.PostgresQuerySample) {
	for _, sample := range inputs {
		// EXPLAIN was already collected, e.g. from auto_explain
		if sample.HasExplain {
			continue
		}

		// To be on the safe side never EXPLAIN a statement that can't be parsed,
		// or multiple statements in one (leading to accidental execution)
		parsetree, err := pg_query.Parse(sample.Query)
		if err == nil && len(parsetree.Statements) == 1 {
			sample.HasExplain = true
			sample.ExplainSource = pganalyze_collector.QuerySample_STATEMENT_LOG_EXPLAIN_SOURCE
			sample.ExplainFormat = pganalyze_collector.QuerySample_JSON_EXPLAIN_FORMAT
			// TODO: Don't run EXPLAIN on queries that start with the collector marker
			err = db.QueryRow(QueryMarkerSQL + "EXPLAIN (VERBOSE, FORMAT JSON) " + sample.Query).Scan(&sample.ExplainOutput)
			if err != nil {
				sample.ExplainError = fmt.Sprintf("%s", err)
			}
		}

		outputs = append(outputs, sample)
	}

	return
}
