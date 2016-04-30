//go:generate msgp

package snapshot

type Explain struct {
	OccurredAt      NullableUnixTimestamp `msg:"occurred_at"`
	NormalizedQuery string                `msg:"normalized_query"`
	Runtime         float64               `msg:"runtime"`
	ExplainOutput   []interface{}         `msg:"explain_output"`
	ExplainError    NullableString        `msg:"explain_error,omitempty"`
}
