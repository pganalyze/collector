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
				SeqTupRead:  stats.SeqTupRead,
				IdxScan:     stats.IdxScan,
				IdxTupFetch: stats.IdxTupFetch,
				NTupIns:     stats.NTupIns,
				NTupUpd:     stats.NTupUpd,
				NTupDel:     stats.NTupDel,
				NTupHotUpd:  stats.NTupHotUpd,
				NLiveTup:    stats.NLiveTup,
				NDeadTup:    stats.NDeadTup,
				//NModSinceAnalyze: stats.NModSinceAnalyze, // FIXME
				HeapBlksRead:  stats.HeapBlksRead,
				HeapBlksHit:   stats.HeapBlksHit,
				IdxBlksRead:   stats.IdxBlksRead,
				IdxBlksHit:    stats.IdxBlksHit,
				ToastBlksRead: stats.ToastBlksRead,
				ToastBlksHit:  stats.ToastBlksHit,
				TidxBlksRead:  stats.TidxBlksRead,
				TidxBlksHit:   stats.TidxBlksHit,
			}
			s.RelationStatistics = append(s.RelationStatistics, &statistic)
		}

		// TODO: Events
	}

	return s
}
