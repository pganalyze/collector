package state

import "github.com/guregu/null"

// PostgresServerStats - Statistics for a Postgres server.
type PostgresServerStats struct {
	CurrentXactId   Xid8
	NextMultiXactId Xid8

	XminHorizonBackend                Xid
	XminHorizonReplicationSlot        Xid
	XminHorizonReplicationSlotCatalog Xid
	XminHorizonPreparedXact           Xid
	XminHorizonStandby                Xid
}

// FullXminHorizonBackend - Returns XminHorizonBackend in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonBackend() int64 {
	return int64(XidToXid8(ss.XminHorizonBackend, Xid8(ss.CurrentXactId)))
}

// FullXminHorizonReplicationSlot - Returns XminHorizonReplicationSlot in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonReplicationSlot() int64 {
	return int64(XidToXid8(ss.XminHorizonReplicationSlot, Xid8(ss.CurrentXactId)))
}

// FullXminHorizonReplicationSlotCatalog - Returns XminHorizonReplicationSlotCatalog in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonReplicationSlotCatalog() int64 {
	return int64(XidToXid8(ss.XminHorizonReplicationSlotCatalog, Xid8(ss.CurrentXactId)))
}

// FullXminHorizonPreparedXact - Returns XminHorizonPreparedXact in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonPreparedXact() int64 {
	return int64(XidToXid8(ss.XminHorizonPreparedXact, Xid8(ss.CurrentXactId)))
}

// FullXminHorizonStandby - Returns XminHorizonStandby in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonStandby() int64 {
	return int64(XidToXid8(ss.XminHorizonStandby, Xid8(ss.CurrentXactId)))
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
