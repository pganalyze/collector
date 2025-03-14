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

type PgStatStatementsStats struct {
	Dealloc int64
	Reset   null.Time
}

type DiffedPgStatStatementsStats PgStatStatementsStats

func (curr PgStatStatementsStats) DiffSince(prev PgStatStatementsStats) DiffedPgStatStatementsStats {
	return DiffedPgStatStatementsStats{
		Dealloc: curr.Dealloc - prev.Dealloc,
		Reset:   curr.Reset,
	}
}

type PostgresServerIoStatsKey struct {
	BackendType string // a backend type like "autovacuum worker"
	IoObject    string // "relation" or "temp relation"
	IoContext   string // "normal", "vacuum", "bulkread" or "bulkwrite"
}

type PostgresServerIoStats struct {
	Reads         int64
	ReadTime      float64
	Writes        int64
	WriteTime     float64
	Writebacks    int64
	WritebackTime float64
	Extends       int64
	ExtendTime    float64
	Hits          int64
	Evictions     int64
	Reuses        int64
	Fsyncs        int64
	FsyncTime     float64
}

type PostgresServerIoStatsMap map[PostgresServerIoStatsKey]PostgresServerIoStats

type DiffedPostgresServerIoStats PostgresServerIoStats
type DiffedPostgresServerIoStatsMap map[PostgresServerIoStatsKey]DiffedPostgresServerIoStats

type HistoricPostgresServerIoStatsMap map[HistoricStatsTimeKey]DiffedPostgresServerIoStatsMap

func (curr PostgresServerIoStats) DiffSince(prev PostgresServerIoStats) DiffedPostgresServerIoStats {
	diff := DiffedPostgresServerIoStats{}
	diff.Reads = curr.Reads - prev.Reads
	diff.ReadTime = curr.ReadTime - prev.ReadTime
	diff.Writes = curr.Writes - prev.Writes
	diff.WriteTime = curr.WriteTime - prev.WriteTime
	diff.Writebacks = curr.Writebacks - prev.Writebacks
	diff.WritebackTime = curr.WritebackTime - prev.WritebackTime
	diff.Extends = curr.Extends - prev.Extends
	diff.ExtendTime = curr.ExtendTime - prev.ExtendTime
	diff.Hits = curr.Hits - prev.Hits
	diff.Evictions = curr.Evictions - prev.Evictions
	diff.Reuses = curr.Reuses - prev.Reuses
	diff.Fsyncs = curr.Fsyncs - prev.Fsyncs
	diff.FsyncTime = curr.FsyncTime - prev.FsyncTime
	return diff
}
