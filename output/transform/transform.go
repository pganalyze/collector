package transform

import (
	"github.com/pganalyze/collector/output/snapshot"
	"github.com/pganalyze/collector/state"
)

func StateToSnapshot(newState state.State, diffState state.DiffState) snapshot.Snapshot {
	var s snapshot.Snapshot

	s = transformStatements(s, newState, diffState)
	s = transformRelations(s, newState, diffState)

	return s
}
