package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func StateToSnapshot(newState state.State, diffState state.DiffState) snapshot.Snapshot {
	var s snapshot.Snapshot

	s = transformStatements(s, newState, diffState)
	s = transformRelations(s, newState, diffState)

	return s
}
