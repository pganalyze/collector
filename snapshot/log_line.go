//go:generate msgp

package snapshot

// LogLine - "Line" in the PostgreSQL logs - can be multiple lines if they belong together
type LogLine struct {
	OccurredAt      NullableUnixTimestamp `msg:"occurred_at"`
	Source          SourceType            `msg:"source"`
	ClientIP        string                `msg:"client_ip,omitempty"`
	LogLevel        string                `msg:"log_level"`
	BackendPid      int                   `msg:"backend_pid"`
	Content         string                `msg:"content"`
	AdditionalLines []LogLine             `msg:"additional_lines,omitempty"`
}

// SourceType - Enum that describes the source of the log line
type SourceType int

// Treat this list as append-only and never change the order
const (
	PostgresSource  SourceType = iota // PostgreSQL server log
	AmazonRdsSource                   // Amazon RDS system logs (backups, restarts, etc)
)
