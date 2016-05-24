package state

import "gopkg.in/guregu/null.v3"

// PostgresBackend - PostgreSQL server backend thats currently working, waiting
// or idling (also known as an open connection)
type PostgresBackend struct {
	Pid             int         `json:"pid"`
	Username        null.String `json:"username"`
	ApplicationName null.String `json:"application_name"`
	ClientAddr      null.String `json:"client_addr"`
	BackendStart    null.Time   `json:"backend_start"`
	XactStart       null.Time   `json:"xact_start"`
	QueryStart      null.Time   `json:"query_start"`
	StateChange     null.Time   `json:"state_change"`
	Waiting         null.Bool   `json:"waiting"`
	State           null.String `json:"state"`
	NormalizedQuery null.String `json:"normalized_query"`
}
