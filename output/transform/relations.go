package transform

import (
	"github.com/pganalyze/collector/output/snapshot"
	"github.com/pganalyze/collector/state"
)

func transformRelations(s snapshot.Snapshot, newState state.State, diffState state.DiffState) snapshot.Snapshot {
	for _, relation := range newState.Relations {
		ref := snapshot.RelationReference{
			DatabaseIdx:  0,
			SchemaName:   relation.SchemaName,
			RelationName: relation.RelationName,
		}
		idx := int32(len(s.RelationReferences))
		s.RelationReferences = append(s.RelationReferences, &ref)

		// Information
		info := snapshot.RelationInformation{
			RelationRef:  idx,
			RelationType: relation.RelationType,
		}
		if relation.ViewDefinition != "" {
			info.ViewDefinition = &snapshot.NullString{Valid: true, Value: relation.ViewDefinition}
		}
		// TODO: Add columns and constraints here
		s.RelationInformations = append(s.RelationInformations, &info)

		// Statistic
		stats, exists := diffState.RelationStats[relation.Oid]
		if exists {
			statistic := snapshot.RelationStatistic{
				RelationRef: idx,
				SizeBytes:   stats.SizeBytes,
				SeqScan:     stats.SeqScan,
				NTupUpd:     stats.NTupUpd,
			}
			// TODO: Complete set of stats
			s.RelationStatistics = append(s.RelationStatistics, &statistic)
		}
	}

	return s
}
