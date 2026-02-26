package transform

import (
	"sort"
	"time"

	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func groupStatements(statements state.PostgresStatementMap, statsMap state.DiffedPostgresStatementStatsMap, server *state.Server) map[statementKey]statementValue {
	groupedStatements := make(map[statementKey]statementValue)

	for sKey, stats := range statsMap {
		statement, exist := statements[sKey]
		if !exist {
			fingerprint, exists := server.Fingerprints.Load(sKey.QueryID)
			if exists {
				statement = state.PostgresStatement{QueryTextUnavailable: true, Fingerprint: fingerprint}
			} else {
				statement = state.PostgresStatement{QueryTextUnavailable: true, Fingerprint: util.FingerprintText(util.QueryTextUnavailable)}
			}
		}

		// Note we intentionally don't include sKey.TopLevel here, since we don't (yet)
		// separate statistics based on that attribute in the pganalyze app
		// As such, pg_stat_statements entries that only differ by toplevel will be aggregated here
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

type queryIDKey struct {
	databaseOid state.Oid
	userOid     state.Oid
	queryID     int64
}
type QueryIDKeyToIdx map[queryIDKey]int32

func transformPostgresStatements(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState, roleOidToIdx OidToIdx, databaseOidToIdx OidToIdx, server *state.Server) (snapshot.FullSnapshot, QueryIDKeyToIdx) {
	var queryStats []*snapshot.HistoricQueryStatistics
	queryIDKeyToIDx := make(QueryIDKeyToIdx)

	for timeKey, diffedStats := range transientState.StatementStats {
		// Ignore any data older than an hour, as a safety measure in case of many
		// failed full snapshot runs (which don't reset state)
		if time.Since(timeKey.CollectedAt).Hours() >= 1 {
			continue
		}

		h := snapshot.HistoricQueryStatistics{}
		h.CollectedAt = timestamppb.New(timeKey.CollectedAt)
		h.CollectedIntervalSecs = timeKey.CollectedIntervalSecs

		groupedStatements := groupStatements(transientState.Statements, diffedStats, server)
		for key, value := range groupedStatements {
			idx := upsertQueryReferenceAndInformation(&s, transientState.StatementTexts, roleOidToIdx, databaseOidToIdx, key, value)
			// Store the map of QueryIdx (idx here) and databaseOid, userOid, queryID combinations
			// to use them later on with plans transformation
			for _, queryId := range value.queryIDs {
				sKey := queryIDKey{
					databaseOid: key.databaseOid,
					userOid:     key.userOid,
					queryID:     queryId,
				}
				queryIDKeyToIDx[sKey] = idx
			}
			if value.statement.IgnoreIoTiming {
				value.statementStats.BlkReadTime = 0
				value.statementStats.BlkWriteTime = 0
			}
			statistic := transformQueryStatistic(value.statementStats, idx)
			h.Statistics = append(h.Statistics, &statistic)
		}
		queryStats = append(queryStats, &h)
	}

	if len(queryStats) == 0 {
		return s, queryIDKeyToIDx
	}

	// Sort the data so we can reliably store the most recent one separately
	sort.Slice(queryStats, func(i, j int) bool {
		return queryStats[i].CollectedAt.Seconds < queryStats[j].CollectedAt.Seconds
	})

	s.QueryStatistics = queryStats[len(queryStats)-1].Statistics
	s.HistoricQueryStatistics = queryStats[:len(queryStats)-1]

	return s, queryIDKeyToIDx
}
