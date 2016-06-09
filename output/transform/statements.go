package transform

import (
	"github.com/pganalyze/collector/output/snapshot"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func upsertQueryReference(s *snapshot.Snapshot, databaseIdx int32, userIdx int32, fingerprint []byte) int32 {
	ref := snapshot.QueryReference{
		DatabaseIdx: databaseIdx,
		UserIdx:     userIdx,
		Fingerprint: fingerprint,
	}

	idx := int32(len(s.QueryReferences))
	s.QueryReferences = append(s.QueryReferences, &ref)

	return idx
}

func transformStatements(s snapshot.Snapshot, newState state.State, diffState state.DiffState) snapshot.Snapshot {
	// TODO: Group together queries by fingerprint

	for _, statement := range diffState.Statements {
		databaseIdx := int32(0) // FIXME
		userIdx := int32(0)     // FIXME
		fingerprint := util.FingerprintQuery(statement.NormalizedQuery)
		idx := upsertQueryReference(&s, databaseIdx, userIdx, fingerprint)

		// Information
		queryInformation := snapshot.QueryInformation{
			QueryRef:        idx,
			NormalizedQuery: statement.NormalizedQuery,
		}
		s.QueryInformations = append(s.QueryInformations, &queryInformation)

		// Statistic
		statistic := snapshot.QueryStatistic{
			QueryRef: idx,

			Calls:             statement.Calls,
			TotalTime:         statement.TotalTime,
			Rows:              statement.Rows,
			SharedBlksHit:     statement.SharedBlksHit,
			SharedBlksRead:    statement.SharedBlksRead,
			SharedBlksDirtied: statement.SharedBlksDirtied,
			SharedBlksWritten: statement.SharedBlksWritten,
			LocalBlksHit:      statement.LocalBlksHit,
			LocalBlksRead:     statement.LocalBlksRead,
			LocalBlksDirtied:  statement.LocalBlksDirtied,
			LocalBlksWritten:  statement.LocalBlksWritten,
			TempBlksRead:      statement.TempBlksRead,
			TempBlksWritten:   statement.TempBlksWritten,
			BlkReadTime:       statement.BlkReadTime,
			BlkWriteTime:      statement.BlkWriteTime,
		}

		if statement.MinTime.Valid {
			statistic.MinTime = statement.MinTime.Float64
		}

		if statement.MaxTime.Valid {
			statistic.MaxTime = statement.MaxTime.Float64
		}

		if statement.MeanTime.Valid {
			statistic.MeanTime = statement.MeanTime.Float64
		}

		if statement.StddevTime.Valid {
			statistic.StddevTime = statement.StddevTime.Float64
		}

		s.QueryStatistics = append(s.QueryStatistics, &statistic)
	}

	return s
}
