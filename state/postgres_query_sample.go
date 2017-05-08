package state

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type PostgresQuerySample struct {
	OccurredAt time.Time
	Username   string
	Database   string
	Query      string

	LogLineUUID uuid.UUID

	RuntimeMs float64

	HasExplain    bool
	ExplainOutput string
	ExplainError  string

	//Parameters []string // TODO: Not implemented

	// FUTURE: Could use parameters (and query values) to determine whether
	// the given value is included in most_common_vals (and which most_common_freqs it has)
}
