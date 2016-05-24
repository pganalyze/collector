package state

import "time"

type PostgresExplain struct {
	OccurredAt      time.Time     `json:"occurred_at"`
	NormalizedQuery string        `json:"normalized_query"`
	Fingerprint     []byte        `json:"fingerprint"`
	Runtime         float64       `json:"runtime"`
	ExplainOutput   []interface{} `json:"explain_output"`
	ExplainError    *string       `json:"explain_error,omitempty"`
}

type PostgresExplainInput struct {
	OccurredAt time.Time
	Query      string
	Runtime    float64
	Parameters []string // TODO: Not implemented

	// FUTURE: Could use parameters (and query values) to determine whether
	// the given value is included in most_common_vals (and which most_common_freqs it has)
}
