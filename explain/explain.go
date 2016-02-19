package explain

import (
	"database/sql"
	"encoding/json"
	"fmt"

	pg_query "github.com/lfittl/pg_query.go"
	"github.com/lfittl/pganalyze-collector-next/util"
)

type ExplainInput struct {
	OccurredAt util.Timestamp
	Query      string
	Runtime    float64
	Parameters []string // TODO: Not implemented

	// FUTURE: Could use parameters (and query values) to determine whether
	// the given value is included in most_common_vals (and which most_common_freqs it has)
}

type Explain struct {
	OccurredAt      util.Timestamp `json:"occurred_at"`
	NormalizedQuery string         `json:"normalized_query"`
	Runtime         float64        `json:"runtime"`
	ExplainOutput   []interface{}  `json:"explain_output"`
	ExplainError    *string        `json:"explain_error,omitempty"`
}

func RunExplain(db *sql.DB, inputs []ExplainInput) (explains []Explain) {
	for _, input := range inputs {
		// To be on the safe side never EXPLAIN a statement that can't be parsed,
		// or multiple statements in one (leading to accidental execution)
		parsetree, err := pg_query.Parse(input.Query)
		if err != nil && len(parsetree.Statements) != 1 {
			continue
		}

		normalizedQuery, err := pg_query.Normalize(input.Query)
		if err != nil {
			continue
		}

		explainOut := Explain{
			OccurredAt:      input.OccurredAt,
			NormalizedQuery: normalizedQuery,
			Runtime:         input.Runtime,
		}

		var planStr []byte
		err = db.QueryRow("EXPLAIN (VERBOSE, FORMAT JSON) " + input.Query).Scan(&planStr)
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
