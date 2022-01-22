package transform

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func groupStatements(statements state.PostgresStatementMap, statementTexts state.PostgresStatementTextMap, statsMap state.DiffedPostgresStatementStatsMap) map[statementKey]statementValue {
	groupedStatements := make(map[statementKey]statementValue)

	for sKey, stats := range statsMap {
		statement, exist := statements[sKey]
		if !exist {
			statement = state.PostgresStatement{QueryTextUnavailable: true, Fingerprint: util.FingerprintText(util.QueryTextUnavailable)}
		}

		key := statementKey{
			databaseOid: sKey.DatabaseOid,
			userOid:     sKey.UserOid,
			fingerprint: statement.Fingerprint,
		}

		value, exist := groupedStatements[key]
		if exist {
			groupedStatements[key] = statementValue{
				statement:      value.statement,
				statementStats: value.statementStats.Add(stats),
				queryIDs:       append(value.queryIDs, sKey.QueryID),
			}
		} else {
			groupedStatements[key] = statementValue{
				statement:      statement,
				statementStats: stats,
				queryIDs:       []int64{sKey.QueryID},
			}
		}
	}

	return groupedStatements
}

func transformQueryStatistic(stats state.DiffedPostgresStatementStats, idx int32) snapshot.QueryStatistic {
	return snapshot.QueryStatistic{
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
}

func transformPostgresStatements(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx) snapshot.FullSnapshot {
	// Statement stats from this snapshot
	groupedStatements := groupStatements(transientState.Statements, transientState.StatementTexts, diffState.StatementStats)
	for key, value := range groupedStatements {
		idx := upsertQueryReferenceAndInformation(&s, transientState.StatementTexts, roleOidToIdx, databaseOidToIdx, key, value)

		statistic := transformQueryStatistic(value.statementStats, idx)
		s.QueryStatistics = append(s.QueryStatistics, &statistic)
	}

	// Historic statement stats which are sent now since we got the query text only now
	for timeKey, diffedStats := range transientState.HistoricStatementStats {
		// Ignore any data older than an hour, as a safety measure in case of many
		// failed full snapshot runs (which don't reset state)
		if time.Since(timeKey.CollectedAt).Hours() >= 1 {
			continue
		}

		h := snapshot.HistoricQueryStatistics{}
		h.CollectedAt, _ = ptypes.TimestampProto(timeKey.CollectedAt)
		h.CollectedIntervalSecs = timeKey.CollectedIntervalSecs

		groupedStatements = groupStatements(transientState.Statements, transientState.StatementTexts, diffedStats)
		for key, value := range groupedStatements {
			idx := upsertQueryReferenceAndInformation(&s, transientState.StatementTexts, roleOidToIdx, databaseOidToIdx, key, value)
			statistic := transformQueryStatistic(value.statementStats, idx)
			h.Statistics = append(h.Statistics, &statistic)
		}
		s.HistoricQueryStatistics = append(s.HistoricQueryStatistics, &h)
	}

	return s
}
