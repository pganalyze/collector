package logs

import (
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/explain"
	"github.com/pganalyze/collector/util"
)

// Line - "Line" in the PostgreSQL logs - can be multiple lines if they belong together
type Line struct {
	OccurredAt      util.Timestamp `json:"occurred_at"`
	Source          SourceType     `json:"source"`
	ClientIP        string         `json:"client_ip,omitempty"`
	LogLevel        string         `json:"log_level"`
	BackendPid      int            `json:"backend_pid"`
	Content         string         `json:"content"`
	AdditionalLines []Line         `json:"additional_lines,omitempty"`
}

// SourceType - Enum that describes the source of the log line
type SourceType int

// Treat this list as append-only and never change the order
const (
	PostgresSource  SourceType = iota // PostgreSQL server log
	AmazonRdsSource                   // Amazon RDS system logs (backups, restarts, etc)
)

// GetLogLines - Retrieves all new log lines for this system and returns them
func GetLogLines(config config.DatabaseConfig) (lines []Line, explainInputs []explain.ExplainInput) {
	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceId != "" {
		lines, explainInputs = getFromAmazonRds(config)
	}

	return
}
