package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func upsertQueryReference(s *snapshot.Snapshot, ref *snapshot.QueryReference) int32 {
	idx := int32(len(s.QueryReferences))
	s.QueryReferences = append(s.QueryReferences, ref)
	return idx
}

type statementKey struct {
	databaseOid state.Oid
	userOid     state.Oid
	fingerprint [21]byte
}

type statementValue struct {
	queryIDs  []int64
	statement state.DiffedPostgresStatement
}

func groupStatements(statements []state.DiffedPostgresStatement) map[statementKey]statementValue {
	groupedStatements := make(map[statementKey]statementValue)

	for _, statement := range statements {
		key := statementKey{
			databaseOid: statement.DatabaseOid,
			userOid:     statement.UserOid,
			fingerprint: util.FingerprintQuery(statement.NormalizedQuery),
		}

		value, exist := groupedStatements[key]
		if exist {
			groupedStatements[key] = statementValue{
				statement: value.statement.Add(statement),
				queryIDs:  append(value.queryIDs, statement.QueryID.Int64),
			}
		} else {
			groupedStatements[key] = statementValue{
				statement: statement,
				queryIDs:  []int64{statement.QueryID.Int64},
			}
		}
	}

	return groupedStatements
}

func transformStatements(s snapshot.Snapshot, newState state.State, diffState state.DiffState) snapshot.Snapshot {
	groupedStatements := groupStatements(diffState.Statements)

	for key, value := range groupedStatements {
		// Note: For whichever reason, we need to use a separate variable here so each fingerprint
		// gets its own memory location (otherwise they're all the one of the last fingerprint value)
		fp := key.fingerprint

		ref := snapshot.QueryReference{
			DatabaseIdx: int32(0), // FIXME
			UserIdx:     int32(0), // FIXME
			Fingerprint: fp[:],
		}
		idx := upsertQueryReference(&s, &ref)

		statement := value.statement

		// Information
		queryInformation := snapshot.QueryInformation{
			QueryRef:        idx,
			NormalizedQuery: statement.NormalizedQuery,
			QueryIds:        value.queryIDs,
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

		s.QueryStatistics = append(s.QueryStatistics, &statistic)
	}

	return s
}
