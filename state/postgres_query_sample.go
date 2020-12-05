package state

import (
	"time"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	uuid "github.com/satori/go.uuid"
)

type PostgresQuerySample struct {
	OccurredAt time.Time
	Username   string
	Database   string
	Query      string
	Parameters []null.String

	LogLineUUID uuid.UUID

	RuntimeMs float64

	HasExplain    bool
	ExplainOutput string
	ExplainError  string
	ExplainFormat pganalyze_collector.QuerySample_ExplainFormat
	ExplainSource pganalyze_collector.QuerySample_ExplainSource

	// FUTURE: Could use parameters (and query values) to determine whether
	// the given value is included in most_common_vals (and which most_common_freqs it has)
}
