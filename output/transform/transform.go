package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func StateToSnapshot(newState state.PersistedState, diffState state.DiffState, transientState state.TransientState) snapshot.FullSnapshot {
	var s snapshot.FullSnapshot

	s = transformPostgres(s, newState, diffState, transientState)
	s = transformSystem(s, newState, diffState)
	s = transformCollectorStats(s, newState, diffState)

	return s
}
