package transform

import (
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/pganalyze/collector/input/postgres"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func ignoredStatement(query string) bool {
	return strings.HasPrefix(query, postgres.QueryMarkerSQL) || strings.HasPrefix(query, "DEALLOCATE") || query == "<insufficient privilege>"
}

func groupStatements(statements state.PostgresStatementMap, statsMap state.DiffedPostgresStatementStatsMap, explainMap state.PostgresStatementExplainMap) map[statementKey]statementValue {
	groupedStatements := make(map[statementKey]statementValue)

	for sKey, stats := range statsMap {
		statement, exist := statements[sKey]
		if !exist {
			statement = state.PostgresStatement{NormalizedQuery: "<unidentified queryid>"}
		} else if ignoredStatement(statement.NormalizedQuery) {
			continue
		}

		key := statementKey{
			databaseOid: sKey.DatabaseOid,
			userOid:     sKey.UserOid,
			fingerprint: util.FingerprintQuery(statement.NormalizedQuery),
		}

		value, exist := groupedStatements[key]
		if exist {
			newValue := statementValue{
				statement:      value.statement,
				statementStats: value.statementStats.Add(stats),
				queryIDs:       append(value.queryIDs, sKey.QueryID),
			}
			explain, exist := explainMap[sKey]
			if exist {
				newValue.explains = append(newValue.explains, snapshot.QueryExplainInformation{
					ExplainOutput: explain.ExplainOutput,
					ExplainError:  explain.ExplainError,
					ExplainFormat: explain.ExplainFormat,
					ExplainSource: explain.ExplainSource,
				})
			}
			groupedStatements[key] = newValue
		} else {
			newValue := statementValue{
				statement:      statement,
				statementStats: stats,
				queryIDs:       []int64{sKey.QueryID},
			}
			explain, exist := explainMap[sKey]
			if exist {
				newValue.explains = []snapshot.QueryExplainInformation{snapshot.QueryExplainInformation{
					ExplainOutput: explain.ExplainOutput,
					ExplainError:  explain.ExplainError,
					ExplainFormat: explain.ExplainFormat,
					ExplainSource: explain.ExplainSource,
				}}
			}
			groupedStatements[key] = newValue
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
	groupedStatements := groupStatements(transientState.Statements, diffState.StatementStats, transientState.StatementExplains)
	for key, value := range groupedStatements {
		idx := upsertQueryReferenceAndInformation(&s, roleOidToIdx, databaseOidToIdx, key, value)

		statistic := transformQueryStatistic(value.statementStats, idx)
		s.QueryStatistics = append(s.QueryStatistics, &statistic)

		for _, explain := range value.explains {
			explain.QueryIdx = idx
			s.QueryExplains = append(s.QueryExplains, &explain)
		}
	}

	// Historic statement stats which are sent now since we got the query text only now
	for timeKey, diffedStats := range transientState.HistoricStatementStats {
		h := snapshot.HistoricQueryStatistics{}
		h.CollectedAt, _ = ptypes.TimestampProto(timeKey.CollectedAt)
		h.CollectedIntervalSecs = timeKey.CollectedIntervalSecs

		groupedStatements = groupStatements(transientState.Statements, diffedStats, nil)
		for key, value := range groupedStatements {
			idx := upsertQueryReferenceAndInformation(&s, roleOidToIdx, databaseOidToIdx, key, value)
			statistic := transformQueryStatistic(value.statementStats, idx)
			h.Statistics = append(h.Statistics, &statistic)
		}
		s.HistoricQueryStatistics = append(s.HistoricQueryStatistics, &h)
	}

	return s
}
