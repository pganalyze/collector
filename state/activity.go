package state

import "time"

type TransientActivityState struct {
	CollectedAt time.Time

	Version  PostgresVersion
	Backends []PostgresBackend

	Vacuums []PostgresVacuumProgress
}

type PersistedActivityState struct {
	ActivitySnapshotAt time.Time
}