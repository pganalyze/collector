package explain

import (
	"database/sql"
	"fmt"

	pg_query "github.com/lfittl/pg_query.go"
)

type ExplainInput struct {
	Query      string
	Runtime    float64
	Parameters []string // TODO: Not implemented

	// FUTURE: Could use parameters (and query values) to determine whether
	// the given value is included in most_common_vals (and which most_common_freqs it has)
}

type Explain struct {
	NormalizedQuery string
	Runtime         float64
	PlanOutput      string
}

func RunExplain(db *sql.DB, inputs []ExplainInput) (explains []Explain) {
	for _, input := range inputs {
		fmt.Printf("%s\n", input.Query)

		// To be on the safe side never EXPLAIN a statement that can't be parsed,
		// or multiple statements in one (leading to accidental execution)
		parsetree, err := pg_query.Parse(input.Query)
		if err != nil && len(parsetree.Statements) != 1 {
			continue
		}

		explainOut := Explain{
			NormalizedQuery: input.Query,
			Runtime:         input.Runtime,
		}

		err = db.QueryRow("EXPLAIN (VERBOSE, FORMAT JSON) " + input.Query).Scan(&explainOut.PlanOutput)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}

		explains = append(explains, explainOut)
	}

	return
}
