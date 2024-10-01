package state

import "time"

// PostgresVacuumProgress - PostgreSQL vacuum thats currently running
//
// See https://www.postgresql.org/docs/current/progress-reporting.html
type PostgresVacuumProgress struct {
	VacuumIdentity  uint64 // Combination of vacuum "query" start time and PID, used to identify a vacuum over time
	BackendIdentity uint64 // Combination of process start time and PID, used to identify a process over time

	DatabaseName string
	SchemaName   string
	RelationName string
	RoleName     string
	StartedAt    time.Time
	Autovacuum   bool
	Toast        bool

	Phase             string
	HeapBlksTotal     int64
	HeapBlksScanned   int64
	HeapBlksVacuumed  int64
	IndexVacuumCount  int64
	MaxDeadItemIds    int64
	NumDeadItemIds    int64
	DeadTupleBytes    int64
	MaxDeadTupleBytes int64
	IndexesTotal      int64
	IndexesProcessed  int64
}
