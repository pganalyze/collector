package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func upsertQueryReference(s *snapshot.FullSnapshot, ref *snapshot.QueryReference) int32 {
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
	queryIDs       []int64
	statement      state.PostgresStatement
	statementKey   state.PostgresStatementKey
	statementStats state.DiffedPostgresStatementStats
}

func groupStatements(statements state.PostgresStatementMap, statsMap state.DiffedPostgresStatementStatsMap) map[statementKey]statementValue {
	groupedStatements := make(map[statementKey]statementValue)

	for sKey, statement := range statements {
		// For now we don't want statements without statistics (first run, consecutive runs with no change)
		stats, exist := statsMap[sKey]
		if !exist {
			continue
		}

		key := statementKey{
			databaseOid: sKey.DatabaseOid,
			userOid:     sKey.UserOid,
			fingerprint: util.FingerprintQuery(statement.NormalizedQuery),
		}

		value, exist := groupedStatements[key]
		if exist {
			groupedStatements[key] = statementValue{
				statement:      value.statement,
				statementKey:   value.statementKey,
				statementStats: value.statementStats.Add(stats),
				queryIDs:       append(value.queryIDs, sKey.QueryID),
			}
		} else {
			groupedStatements[key] = statementValue{
				statement:      statement,
				statementKey:   sKey,
				statementStats: stats,
				queryIDs:       []int64{sKey.QueryID},
			}
		}
	}

	return groupedStatements
}

func transformPostgresStatements(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	groupedStatements := groupStatements(transientState.Statements, diffState.StatementStats)

	for key, value := range groupedStatements {
		// Note: For whichever reason, we need to use a separate variable here so each fingerprint
		// gets its own memory location (otherwise they're all the one of the last fingerprint value)
		fp := key.fingerprint

		ref := snapshot.QueryReference{
			DatabaseIdx: databaseOidToIdx[key.databaseOid],
			RoleIdx:     roleOidToIdx[key.userOid],
			Fingerprint: fp[:],
		}
		idx := upsertQueryReference(&s, &ref)

		statement := value.statement
		stats := value.statementStats

		// Information
		queryInformation := snapshot.QueryInformation{
			QueryIdx:        idx,
			NormalizedQuery: statement.NormalizedQuery,
			QueryIds:        value.queryIDs,
		}
		s.QueryInformations = append(s.QueryInformations, &queryInformation)

		// Statistic
		statistic := snapshot.QueryStatistic{
			QueryIdx: idx,

			Calls:             stats.Calls,
			TotalTime:         stats.TotalTime,
			Rows:              stats.Rows,
			SharedBlksHit:     stats.SharedBlksHit,
			SharedBlksRead:    stats.SharedBlksRead,
			SharedBlksDirtied: stats.SharedBlksDirtied,
			SharedBlksWritten: stats.SharedBlksWritten,
			LocalBlksHit:      stats.LocalBlksHit,
			LocalBlksRead:     stats.LocalBlksRead,
			LocalBlksDirtied:  stats.LocalBlksDirtied,
			LocalBlksWritten:  stats.LocalBlksWritten,
			TempBlksRead:      stats.TempBlksRead,
			TempBlksWritten:   stats.TempBlksWritten,
			BlkReadTime:       stats.BlkReadTime,
			BlkWriteTime:      stats.BlkWriteTime,
		}

		s.QueryStatistics = append(s.QueryStatistics, &statistic)
	}

	return s
}
