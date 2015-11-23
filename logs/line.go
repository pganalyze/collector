package logs

import (
	"github.com/lfittl/pganalyze-collector-next/config"
	"github.com/lfittl/pganalyze-collector-next/util"
)

// Line - "Line" in the PostgreSQL logs - can be multiple lines if they belong together
type Line struct {
	OccurredAt util.Timestamp `json:"occurred_at"`
	Content    string         `json:"content"`
	Source     SourceType     `json:"type"`
}

// SourceType - Enum that describes the source of the log line
type SourceType int

// Treat this list as append-only and never change the order
const (
	PostgresSource  SourceType = iota // PostgreSQL server log
	AmazonRdsSource                   // Amazon RDS system logs (backups, restarts, etc)
)

// GetLogLines - Retrieves all new log lines for this system and returns them
func GetLogLines(config config.Config) (lines []Line) {
	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceId != "" {
		lines = getFromAmazonRds(config)
	}

	return
}
