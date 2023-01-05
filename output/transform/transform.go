package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func StateToSnapshot(server *state.Server, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState) snapshot.FullSnapshot {
	var s snapshot.FullSnapshot

	s = transformPostgres(s, server, newState, diffState, transientState)
	s = systemStateToFullSnapshot(s, newState, diffState)
	s = transformCollectorStats(s, newState, diffState)
	s = transformCollectorPlatform(s, transientState)
	s = transformCollectorConfig(s, transientState)

	return s
}
