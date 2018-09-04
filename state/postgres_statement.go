package state

import (
	"time"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/output/pganalyze_collector"
)

// PostgresStatement - Specific kind of statement that has run one or multiple times
// on the PostgreSQL server.
type PostgresStatement struct {
	NormalizedQuery string // Text of a representative statement (normalized)
}

// PostgresStatementStats - Statistics from pg_stat_statements extension for a given
// statement.
//
// See also https://www.postgresql.org/docs/9.5/static/pgstatstatements.html
type PostgresStatementStats struct {
	Calls             int64   // Number of times executed
	TotalTime         float64 // Total time spent in the statement, in milliseconds
	Rows              int64   // Total number of rows retrieved or affected by the statement
	SharedBlksHit     int64   // Total number of shared block cache hits by the statement
	SharedBlksRead    int64   // Total number of shared blocks read by the statement
	SharedBlksDirtied int64   // Total number of shared blocks dirtied by the statement
	SharedBlksWritten int64   // Total number of shared blocks written by the statement
	LocalBlksHit      int64   // Total number of local block cache hits by the statement
	LocalBlksRead     int64   // Total number of local blocks read by the statement
	LocalBlksDirtied  int64   // Total number of local blocks dirtied by the statement
	LocalBlksWritten  int64   // Total number of local blocks written by the statement
	TempBlksRead      int64   // Total number of temp blocks read by the statement
	TempBlksWritten   int64   // Total number of temp blocks written by the statement
	BlkReadTime       float64 // Total time the statement spent reading blocks, in milliseconds (if track_io_timing is enabled, otherwise zero)
	BlkWriteTime      float64 // Total time the statement spent writing blocks, in milliseconds (if track_io_timing is enabled, otherwise zero)

	// Postgres 9.5+
	MinTime    null.Float // Minimum time spent in the statement, in milliseconds
	MaxTime    null.Float // Maximum time spent in the statement, in milliseconds
	MeanTime   null.Float // Mean time spent in the statement, in milliseconds
	StddevTime null.Float // Population standard deviation of time spent in the statement, in milliseconds
}

// PostgresStatementKey - Information that uniquely identifies a query
type PostgresStatementKey struct {
	DatabaseOid Oid   // OID of database in which the statement was executed
	UserOid     Oid   // OID of user who executed the statement
	QueryID     int64 // Postgres 9.4+: Internal hash code, computed from the statement's parse tree
}

type PostgresStatementStatsTimeKey struct {
	CollectedAt           time.Time
	CollectedIntervalSecs uint32
}

type PostgresStatementMap map[PostgresStatementKey]PostgresStatement
type PostgresStatementStatsMap map[PostgresStatementKey]PostgresStatementStats

type DiffedPostgresStatementStats PostgresStatementStats
type DiffedPostgresStatementStatsMap map[PostgresStatementKey]DiffedPostgresStatementStats

type HistoricStatementStatsMap map[PostgresStatementStatsTimeKey]DiffedPostgresStatementStatsMap

type PostgresStatementExplain pganalyze_collector.QueryExplainInformation
type PostgresStatementExplainMap map[PostgresStatementKey]PostgresStatementExplain

func (curr PostgresStatementStats) DiffSince(prev PostgresStatementStats) DiffedPostgresStatementStats {
	return DiffedPostgresStatementStats{
		Calls:             curr.Calls - prev.Calls,
		TotalTime:         curr.TotalTime - prev.TotalTime,
		Rows:              curr.Rows - prev.Rows,
		SharedBlksHit:     curr.SharedBlksHit - prev.SharedBlksHit,
		SharedBlksRead:    curr.SharedBlksRead - prev.SharedBlksRead,
		SharedBlksDirtied: curr.SharedBlksDirtied - prev.SharedBlksDirtied,
		SharedBlksWritten: curr.SharedBlksWritten - prev.SharedBlksWritten,
		LocalBlksHit:      curr.LocalBlksHit - prev.LocalBlksHit,
		LocalBlksRead:     curr.LocalBlksRead - prev.LocalBlksRead,
		LocalBlksDirtied:  curr.LocalBlksDirtied - prev.LocalBlksDirtied,
		LocalBlksWritten:  curr.LocalBlksWritten - prev.LocalBlksWritten,
		TempBlksRead:      curr.TempBlksRead - prev.TempBlksRead,
		TempBlksWritten:   curr.TempBlksWritten - prev.TempBlksWritten,
		BlkReadTime:       curr.BlkReadTime - prev.BlkReadTime,
		BlkWriteTime:      curr.BlkWriteTime - prev.BlkWriteTime,
	}
}

// Add - Adds the statistics of one diffed statement to another, returning the result as a copy
func (stmt DiffedPostgresStatementStats) Add(other DiffedPostgresStatementStats) DiffedPostgresStatementStats {
	return DiffedPostgresStatementStats{
		Calls:             stmt.Calls + other.Calls,
		TotalTime:         stmt.TotalTime + other.TotalTime,
		Rows:              stmt.Rows + other.Rows,
		SharedBlksHit:     stmt.SharedBlksHit + other.SharedBlksHit,
		SharedBlksRead:    stmt.SharedBlksRead + other.SharedBlksRead,
		SharedBlksDirtied: stmt.SharedBlksDirtied + other.SharedBlksDirtied,
		SharedBlksWritten: stmt.SharedBlksWritten + other.SharedBlksWritten,
		LocalBlksHit:      stmt.LocalBlksHit + other.LocalBlksHit,
		LocalBlksRead:     stmt.LocalBlksRead + other.LocalBlksRead,
		LocalBlksDirtied:  stmt.LocalBlksDirtied + other.LocalBlksDirtied,
		LocalBlksWritten:  stmt.LocalBlksWritten + other.LocalBlksWritten,
		TempBlksRead:      stmt.TempBlksRead + other.TempBlksRead,
		TempBlksWritten:   stmt.TempBlksWritten + other.TempBlksWritten,
		BlkReadTime:       stmt.BlkReadTime + other.BlkReadTime,
		BlkWriteTime:      stmt.BlkWriteTime + other.BlkWriteTime,
	}
}
