package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

// Since Postgres reports parent partition tables as being zero-sized,
// this backfills stats for the parent table as a summation of all child tables.
//
// When ignore_schema_regexp is set, GetRelationStats bypasses this for tables not tracked by the collector.
//
// TODO: recursively build up stats when nested partitioning is used
func mergePartitionSizes(s snapshot.FullSnapshot, newState state.PersistedState, ts state.TransientState, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	relIdxToStatsIdx := make(map[int32]int, len(s.RelationStatistics))
	for idx, stat := range s.RelationStatistics {
		relIdxToStatsIdx[stat.RelationIdx] = idx
	}

	for idx, rel := range s.RelationInformations {
		if !rel.HasParentRelation || rel.PartitionBoundary == "" {
			continue
		}
		statIdx, ok := relIdxToStatsIdx[int32(idx)]
		if !ok {
			continue
		}

		stat := s.RelationStatistics[statIdx]
		parent := s.RelationStatistics[rel.ParentRelationIdx]
		parent.NTupIns += stat.NTupIns
		parent.NTupUpd += stat.NTupUpd
		parent.NTupDel += stat.NTupDel
		parent.NTupHotUpd += stat.NTupHotUpd
		parent.NLiveTup += stat.NLiveTup
		parent.NDeadTup += stat.NDeadTup
		parent.SizeBytes += stat.SizeBytes
		parent.ToastSizeBytes += stat.ToastSizeBytes
		parent.CachedDataBytes += stat.CachedDataBytes
		parent.CachedToastBytes += stat.CachedToastBytes
	}

	for idx, info := range s.IndexInformations {
		rel := s.RelationInformations[info.RelationIdx]
		if !rel.HasParentRelation || rel.PartitionBoundary == "" {
			continue
		}
		for parentIdx, pi := range s.IndexInformations {
			if pi.RelationIdx != rel.ParentRelationIdx {
				continue
			}
			if info.IndexType == pi.IndexType && info.IsUnique == pi.IsUnique && intArrayEqual(info.Columns, pi.Columns) {
				stat := s.IndexStatistics[idx]
				parent := s.IndexStatistics[parentIdx]
				parent.SizeBytes += stat.SizeBytes
				parent.IdxScan += stat.IdxScan
				parent.IdxTupRead += stat.IdxTupRead
				parent.IdxTupFetch += stat.IdxTupFetch
				parent.IdxBlksRead += stat.IdxBlksRead
				parent.IdxBlksHit += stat.IdxBlksHit
				parent.CachedBytes += stat.CachedBytes
				break
			}
		}
	}

	return s
}

func intArrayEqual(a []int32, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
