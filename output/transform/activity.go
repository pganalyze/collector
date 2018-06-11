package transform

import (
	"github.com/golang/protobuf/ptypes"
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

	for _, vacuum := range activityState.Vacuums {
		vacuumInfo := snapshot.VacuumProgressInformation{
			VacuumIdentity:  vacuum.VacuumIdentity,
			BackendIdentity: vacuum.BackendIdentity,
			Autovacuum:      vacuum.Autovacuum,
		}

		if vacuum.RoleName != "" {
			vacuumInfo.RoleIdx, r.RoleReferences = upsertRoleReference(r.RoleReferences, vacuum.RoleName)
		} else {
			vacuumInfo.RoleIdx = -1
		}

		vacuumInfo.DatabaseIdx, r.DatabaseReferences = upsertDatabaseReference(r.DatabaseReferences, vacuum.DatabaseName)
		relationRef := snapshot.RelationReference{
			DatabaseIdx:  vacuumInfo.DatabaseIdx,
			SchemaName:   vacuum.SchemaName,
			RelationName: vacuum.RelationName,
		}
		vacuumInfo.RelationIdx = int32(len(r.RelationReferences))
		r.RelationReferences = append(r.RelationReferences, &relationRef)

		vacuumInfo.StartedAt, _ = ptypes.TimestampProto(vacuum.StartedAt)

		s.VacuumProgressInformations = append(s.VacuumProgressInformations, &vacuumInfo)

		if vacuum.Phase != "" {
			vacuumStats := snapshot.VacuumProgressStatistic{
				VacuumIdentity:   vacuum.VacuumIdentity,
				HeapBlksTotal:    vacuum.HeapBlksTotal,
				HeapBlksScanned:  vacuum.HeapBlksScanned,
				HeapBlksVacuumed: vacuum.HeapBlksVacuumed,
				IndexVacuumCount: vacuum.IndexVacuumCount,
				MaxDeadTuples:    vacuum.MaxDeadTuples,
				NumDeadTuples:    vacuum.NumDeadTuples,
			}

			switch vacuum.Phase {
			case "initializing":
				vacuumStats.Phase = snapshot.VacuumProgressStatistic_INITIALIZING
			case "scanning heap":
				vacuumStats.Phase = snapshot.VacuumProgressStatistic_SCAN_HEAP
			case "vacuuming indexes":
				vacuumStats.Phase = snapshot.VacuumProgressStatistic_VACUUM_INDEX
			case "vacuuming heap":
				vacuumStats.Phase = snapshot.VacuumProgressStatistic_VACUUM_HEAP
			case "cleaning up indexes":
				vacuumStats.Phase = snapshot.VacuumProgressStatistic_INDEX_CLEANUP
			case "truncating heap":
				vacuumStats.Phase = snapshot.VacuumProgressStatistic_TRUNCATE
			case "performing final cleanup":
				vacuumStats.Phase = snapshot.VacuumProgressStatistic_FINAL_CLEANUP
			}

			s.VacuumProgressStatistics = append(s.VacuumProgressStatistics, &vacuumStats)
		}
	}

	return s, r
}
