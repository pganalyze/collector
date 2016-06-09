package transform

import (
	"github.com/pganalyze/collector/output/snapshot"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func transformStatements(s snapshot.Snapshot, newState state.State, diffState state.DiffState) snapshot.Snapshot {
	for _, statement := range diffState.Statements {
		ref := snapshot.QueryReference{
			DatabaseIdx: 0, // FIXME
			UserIdx:     0, // FIXME
			Fingerprint: util.FingerprintQuery(statement.NormalizedQuery),
		}
		idx := int32(len(s.QueryReferences))
		s.QueryReferences = append(s.QueryReferences, &ref)

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
			//MinTime:           statement.MinTime, // FIXME
			//MaxTime:           statement.MaxTime, // FIXME
			//MeanTime:          statement.MeanTime, // FIXME
			//StddevTime:        statement.StddevTime, // FIXME
		}

		s.QueryStatistics = append(s.QueryStatistics, &statistic)
	}

	return s
}
