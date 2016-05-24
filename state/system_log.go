package state

import "time"

// Line - "Line" in the PostgreSQL logs - can be multiple lines if they belong together
type LogLine struct {
	OccurredAt      time.Time  `json:"occurred_at"`
	Source          SourceType `json:"source"`
	ClientIP        string     `json:"client_ip,omitempty"`
	LogLevel        string     `json:"log_level"`
	BackendPid      int        `json:"backend_pid"`
	Content         string     `json:"content"`
	AdditionalLines []LogLine  `json:"additional_lines,omitempty"`
}

// SourceType - Enum that describes the source of the log line
type SourceType int

// Treat this list as append-only and never change the order
const (
	PostgresSource  SourceType = iota // PostgreSQL server log
	AmazonRdsSource                   // Amazon RDS system logs (backups, restarts, etc)
)
