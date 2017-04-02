package state

import "time"

// LogLine - "Line" in the PostgreSQL logs - can be multiple lines if they belong together
type LogLine struct {
	OccurredAt time.Time
	Username   string
	Database   string
	Query      string

	ClientHostname string
	ClientPort     int32
	LogLevel       string
	BackendPid     int32

	Content string

	AdditionalLines []LogLine
}
