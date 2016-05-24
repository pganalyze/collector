package state

import "gopkg.in/guregu/null.v3"

// PostgresFunction - Function/Stored Procedure that runs on the PostgreSQL server
type PostgresFunction struct {
	SchemaName      string      `json:"schema_name"`
	FunctionName    string      `json:"function_name"`
	Language        string      `json:"language"`
	Source          string      `json:"source"`
	SourceBin       null.String `json:"source_bin"`
	Config          null.String `json:"config"`
	Arguments       null.String `json:"arguments"`
	Result          null.String `json:"result"`
	Aggregate       bool        `json:"aggregate"`
	Window          bool        `json:"window"`
	SecurityDefiner bool        `json:"security_definer"`
	Leakproof       bool        `json:"leakproof"`
	Strict          bool        `json:"strict"`
	ReturnsSet      bool        `json:"returns_set"`
	Volatile        null.String `json:"volatile"`
}

// PostgresFunctionStats - Statistics about a single PostgreSQL function
//
// Note that this will only be populated when "track_functions" is enabled.
type PostgresFunctionStats struct {
	Calls     null.Int   `json:"calls"`
	TotalTime null.Float `json:"total_time"`
	SelfTime  null.Float `json:"self_time"`
}
