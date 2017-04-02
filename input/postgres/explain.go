package postgres

import (
	"database/sql"
	"fmt"

	pg_query "github.com/lfittl/pg_query_go"
	"github.com/pganalyze/collector/state"
)

func RunExplain(db *sql.DB, inputs []state.PostgresQuerySample) (outputs []state.PostgresQuerySample) {
	for _, sample := range inputs {
		// To be on the safe side never EXPLAIN a statement that can't be parsed,
		// or multiple statements in one (leading to accidental execution)
		parsetree, err := pg_query.Parse(sample.Query)
		if err == nil && len(parsetree.Statements) == 1 {
			sample.HasExplain = true
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
