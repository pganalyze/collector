package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"

	pg_query "github.com/lfittl/pg_query_go"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func RunExplain(db *sql.DB, inputs []state.PostgresExplainInput) (explains []state.PostgresExplain) {
	for _, input := range inputs {
		// To be on the safe side never EXPLAIN a statement that can't be parsed,
		// or multiple statements in one (leading to accidental execution)
		parsetree, err := pg_query.Parse(input.Query)
		if err != nil && len(parsetree.Statements) != 1 {
			continue
		}

		// TODO: Make sure that we also add the normalized query to QueryInformation (in case it doesn't show up in pgss output)
		normalizedQuery, err := pg_query.Normalize(input.Query)
		if err != nil {
			continue
		}

		fingerprint := util.FingerprintQuery(normalizedQuery)

		explainOut := state.PostgresExplain{
			OccurredAt:      input.OccurredAt,
			NormalizedQuery: normalizedQuery,
			Fingerprint:     fingerprint,
			Runtime:         input.Runtime,
		}

		// TODO: Don't run EXPLAIN on queries that start with the marker

		var planStr []byte
		err = db.QueryRow(QueryMarkerSQL + "EXPLAIN (VERBOSE, FORMAT JSON) " + input.Query).Scan(&planStr)
		if err != nil {
			errorStr := fmt.Sprintf("%s", err)
			explainOut.ExplainError = &errorStr
			explains = append(explains, explainOut)
			continue
		}

		err = json.Unmarshal(planStr, &explainOut.ExplainOutput)
		if err != nil {
			errorStr := fmt.Sprintf("%s", err)
			explainOut.ExplainError = &errorStr
			explains = append(explains, explainOut)
			continue
		}

		explains = append(explains, explainOut)
	}

	return
}
