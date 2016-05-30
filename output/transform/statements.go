package transform

import (
	"github.com/pganalyze/collector/output/snapshot"
	"github.com/pganalyze/collector/state"
)

func transformStatements(s snapshot.Snapshot, newState state.State, diffState state.DiffState) snapshot.Snapshot {
	for _, statement := range diffState.Statements {
		ref := snapshot.QueryReference{
			DatabaseIdx: 0,
			UserIdx:     0,
			Fingerprint: statement.Fingerprint,
		}
		idx := int32(len(s.QueryReferences))
		s.QueryReferences = append(s.QueryReferences, &ref)

		// FIXME: This is (very!) incomplete code

		queryInformation := snapshot.QueryInformation{
			QueryRef:        idx,
			NormalizedQuery: statement.NormalizedQuery,
		}

		s.QueryInformations = append(s.QueryInformations, &queryInformation)
	}

	return s
}
