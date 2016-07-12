package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func StateToSnapshot(newState state.State, diffState state.DiffState) snapshot.FullSnapshot {
	var s snapshot.FullSnapshot

	s = transformPostgres(s, newState, diffState)
	s = transformSystem(s, newState, diffState)
	s = transformCollectorStats(s, newState, diffState)

	return s
}
