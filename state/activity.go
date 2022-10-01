package state

import "time"

type TransientActivityState struct {
	CollectedAt time.Time

	TrackActivityQuerySize int

	Version  PostgresVersion
	Backends []PostgresBackend

	Vacuums []PostgresVacuumProgress

	Locks     []PostgresLock
	LocksFull []PostgresLockFull
}

type PersistedActivityState struct {
	ActivitySnapshotAt time.Time
}
