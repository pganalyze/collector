package explain

import (
	"database/sql"
	"fmt"
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
	fmt.Printf("%v\n", inputs[0].Query)
	// TODO
	return
}
