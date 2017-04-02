package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func StateToSnapshot(newState state.PersistedState, diffState state.DiffState, transientState state.TransientState) snapshot.FullSnapshot {
	var s snapshot.FullSnapshot

	s, roleNameToIdx, databaseNameToIdx := transformPostgres(s, newState, diffState, transientState)
	s = transformSystem(s, newState, diffState)
	s = transformSystemLogs(s, transientState, roleNameToIdx, databaseNameToIdx)
	s = transformCollectorStats(s, newState, diffState)

	return s
}
