package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func ActivityStateToCompactActivitySnapshot(activityState state.ActivityState) (snapshot.CompactActivitySnapshot, snapshot.CompactSnapshot_BaseRefs) {
	var s snapshot.CompactActivitySnapshot
	var r snapshot.CompactSnapshot_BaseRefs

	for _, backend := range activityState.Backends {
		b := transformBackendWithoutRefs(backend)

		if backend.RoleName.Valid {
			b.RoleIdx, r.RoleReferences = upsertRoleReference(r.RoleReferences, backend.RoleName.String)
			b.HasRoleIdx = true
		}

		if backend.DatabaseName.Valid {
			b.DatabaseIdx, r.DatabaseReferences = upsertDatabaseReference(r.DatabaseReferences, backend.DatabaseName.String)
			b.HasDatabaseIdx = true
		}

		if backend.Query.Valid {
			b.QueryIdx, r.QueryReferences, r.QueryInformations = upsertQueryReferenceAndInformationSimple(
				r.QueryReferences,
				r.QueryInformations,
				b.RoleIdx,
				b.DatabaseIdx,
				backend.Query.String,
			)
			b.HasQueryIdx = true
			b.QueryText = backend.Query.String
		}

		s.Backends = append(s.Backends, &b)
	}

	return s, r
}
