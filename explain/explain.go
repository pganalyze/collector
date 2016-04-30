//go:generate msgp

package explain

import (
	"database/sql"
	"fmt"

	pg_query "github.com/lfittl/pg_query_go"
	"github.com/pganalyze/collector/dbstats"
	"github.com/pganalyze/collector/snapshot"
	"github.com/pganalyze/collector/util"
)

type ExplainInput struct {
	OccurredAt util.Timestamp
	Query      string
	Runtime    float64
	Parameters []string // TODO: Not implemented

	// FUTURE: Could use parameters (and query values) to determine whether
	// the given value is included in most_common_vals (and which most_common_freqs it has)
}

func RunExplain(db *sql.DB, inputs []ExplainInput) (explains []*snapshot.Explain) {
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

		explainOut := snapshot.Explain{
			OccurredAt:      input.OccurredAt.Time.Unix(),
			NormalizedQuery: normalizedQuery,
			Runtime:         input.Runtime,
		}

		// TODO: Don't run EXPLAIN on queries that start with the marker

		var planStr []byte
		err = db.QueryRow(dbstats.QueryMarkerSQL + "EXPLAIN (VERBOSE, FORMAT JSON) " + input.Query).Scan(&planStr)
		if err != nil {
			errorStr := fmt.Sprintf("%s", err)
			explainOut.ExplainError = &snapshot.NullString{Valid: true, Value: errorStr}
			explains = append(explains, &explainOut)
			continue
		}

		explainOut.ExplainOutput = string(planStr)

		/*err = json.Unmarshal(planStr, &explainOut.ExplainOutput)
		if err != nil {
			errorStr := fmt.Sprintf("%s", err)
			explainOut.ExplainError = &snapshot.NullString{Valid: true, Value: errorStr}
			explains = append(explains, &explainOut)
			continue
		}*/

		explains = append(explains, &explainOut)
	}

	return
}
