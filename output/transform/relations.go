package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
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
		for _, column := range relation.Columns {
			sColumn := snapshot.RelationInformation_Column{
				Name:     column.Name,
				DataType: column.DataType,
				NotNull:  column.NotNull,
				Position: column.Position,
			}
			if column.DefaultValue.Valid {
				sColumn.DefaultValue = &snapshot.NullString{Valid: true, Value: column.DefaultValue.String}
			}
			info.Columns = append(info.Columns, &sColumn)
		}
		for _, constraint := range relation.Constraints {
			sConstraint := snapshot.RelationInformation_Constraint{
				Name:              constraint.Name,
				Type:              constraint.Type,
				ConstraintDef:     constraint.ConstraintDef,
				ForeignUpdateType: constraint.ForeignUpdateType,
				ForeignDeleteType: constraint.ForeignDeleteType,
				ForeignMatchType:  constraint.ForeignMatchType,
			}
			if constraint.ForeignOid != 0 {
				sConstraint.ForeignRelationRef = -1 // FIXME, need to look this up
			}
			for _, column := range constraint.Columns {
				sConstraint.Columns = append(sConstraint.Columns, int32(column))
			}
			for _, column := range constraint.ForeignColumns {
				sConstraint.ForeignColumns = append(sConstraint.ForeignColumns, int32(column))
			}
			info.Constraints = append(info.Constraints, &sConstraint)
		}
		s.RelationInformations = append(s.RelationInformations, &info)

		// Statistic
		stats, exists := diffState.RelationStats[relation.Oid]
		if exists {
			statistic := snapshot.RelationStatistic{
				RelationRef:   idx,
				SizeBytes:     stats.SizeBytes,
				SeqScan:       stats.SeqScan,
				SeqTupRead:    stats.SeqTupRead,
				IdxScan:       stats.IdxScan,
				IdxTupFetch:   stats.IdxTupFetch,
				NTupIns:       stats.NTupIns,
				NTupUpd:       stats.NTupUpd,
				NTupDel:       stats.NTupDel,
				NTupHotUpd:    stats.NTupHotUpd,
				NLiveTup:      stats.NLiveTup,
				NDeadTup:      stats.NDeadTup,
				HeapBlksRead:  stats.HeapBlksRead,
				HeapBlksHit:   stats.HeapBlksHit,
				IdxBlksRead:   stats.IdxBlksRead,
				IdxBlksHit:    stats.IdxBlksHit,
				ToastBlksRead: stats.ToastBlksRead,
				ToastBlksHit:  stats.ToastBlksHit,
				TidxBlksRead:  stats.TidxBlksRead,
				TidxBlksHit:   stats.TidxBlksHit,
			}
			if stats.NModSinceAnalyze.Valid {
				statistic.NModSinceAnalyze = stats.NModSinceAnalyze.Int64
			}
			s.RelationStatistics = append(s.RelationStatistics, &statistic)
		}

		// TODO: Events

		// Indices
		for _, index := range relation.Indices {
			ref := snapshot.IndexReference{
				DatabaseIdx: 0,
				SchemaName:  relation.SchemaName,
				IndexName:   index.Name,
			}
			indexIdx := int32(len(s.IndexReferences))
			s.IndexReferences = append(s.IndexReferences, &ref)

			// Information
			indexInfo := snapshot.IndexInformation{
				IndexRef:    idx,
				RelationRef: indexIdx,
				IndexDef:    index.IndexDef,
				IsPrimary:   index.IsPrimary,
				IsUnique:    index.IsUnique,
				IsValid:     index.IsValid,
			}
			if index.ConstraintDef.Valid {
				indexInfo.ConstraintDef = &snapshot.NullString{Valid: true, Value: index.ConstraintDef.String}
			}
			for _, column := range index.Columns {
				indexInfo.Columns = append(indexInfo.Columns, int32(column))
			}
			s.IndexInformations = append(s.IndexInformations, &indexInfo)

			// Statistic
			indexStats, exists := diffState.IndexStats[index.IndexOid]
			if exists {
				statistic := snapshot.IndexStatistic{
					IndexRef:    indexIdx,
					SizeBytes:   indexStats.SizeBytes,
					IdxScan:     indexStats.IdxScan,
					IdxTupRead:  indexStats.IdxTupRead,
					IdxTupFetch: indexStats.IdxTupFetch,
					IdxBlksRead: indexStats.IdxBlksRead,
					IdxBlksHit:  indexStats.IdxBlksHit,
				}
				s.IndexStatistics = append(s.IndexStatistics, &statistic)
			}
		}
	}

	return s
}
