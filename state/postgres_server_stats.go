package state

import "github.com/guregu/null"

// PostgresServerStats - Statistics for a Postgres server.
type PostgresServerStats struct {
	CurrentXactId   Xid8
	NextMultiXactId Xid8
}

type PostgresServerIoStatsKey struct {
	BackendType string // a backend type like "autovacuum worker"
	IoObject    string // "relation" or "temp relation"
	IoContext   string // "normal", "vacuum", "bulkread" or "bulkwrite"
}

type PostgresServerIoStats struct {
	Reads     null.Int
	Writes    null.Int
	Extends   null.Int
	OpBytes   int64
	Evictions null.Int
	Reuses    null.Int
	Fsyncs    null.Int
}

type PostgresServerIoStatsMap map[PostgresServerIoStatsKey]PostgresServerIoStats

type DiffedPostgresServerIoStats PostgresServerIoStats
type DiffedPostgresServerIoStatsMap map[PostgresServerIoStatsKey]DiffedPostgresServerIoStats

type HistoricPostgresServerIoStatsMap map[PostgresStatementStatsTimeKey]DiffedPostgresServerIoStatsMap

func (curr PostgresServerIoStats) DiffSince(prev PostgresServerIoStats) DiffedPostgresServerIoStats {
	diff := DiffedPostgresServerIoStats{OpBytes: curr.OpBytes}
	if curr.Reads.Valid && prev.Reads.Valid {
		diff.Reads = null.IntFrom(curr.Reads.Int64 - prev.Reads.Int64)
	}
	if curr.Writes.Valid && prev.Writes.Valid {
		diff.Writes = null.IntFrom(curr.Writes.Int64 - prev.Writes.Int64)
	}
	if curr.Extends.Valid && prev.Extends.Valid {
		diff.Extends = null.IntFrom(curr.Extends.Int64 - prev.Extends.Int64)
	}
	if curr.Evictions.Valid && prev.Evictions.Valid {
		diff.Evictions = null.IntFrom(curr.Evictions.Int64 - prev.Evictions.Int64)
	}
	if curr.Reuses.Valid && prev.Reuses.Valid {
		diff.Reuses = null.IntFrom(curr.Reuses.Int64 - prev.Reuses.Int64)
	}
	if curr.Fsyncs.Valid && prev.Fsyncs.Valid {
		diff.Fsyncs = null.IntFrom(curr.Fsyncs.Int64 - prev.Fsyncs.Int64)
	}
	return diff
}
