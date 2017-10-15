package state

import "time"

type ActivityState struct {
	CollectedAt time.Time

	Version  PostgresVersion
	Backends []PostgresBackend

	Vacuums []PostgresVacuumProgress
}
