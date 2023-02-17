package state

import (
	"database/sql"
)

// PostgresServerStats - Statistics for a Postgres server.
type PostgresServerStats struct {
	CurrentXactId   Xid8
	NextMultiXactId Xid8

	XminHorizonBackend         sql.NullInt32
	XminHorizonReplicationSlot sql.NullInt32
	XminHorizonPreparedXact    sql.NullInt32
	XminHorizonStandby         sql.NullInt32
}

// FullXminHorizonBackend - Returns XminHorizonBackend in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonBackend() int64 {
	return int64(XidToXid8(Xid(ss.XminHorizonBackend.Int32), Xid8(ss.CurrentXactId)))
}

// FullXminHorizonReplicationSlot - Returns XminHorizonReplicationSlot in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonReplicationSlot() int64 {
	return int64(XidToXid8(Xid(ss.XminHorizonReplicationSlot.Int32), Xid8(ss.CurrentXactId)))
}

// FullXminHorizonPreparedXact - Returns XminHorizonPreparedXact in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonPreparedXact() int64 {
	return int64(XidToXid8(Xid(ss.XminHorizonPreparedXact.Int32), Xid8(ss.CurrentXactId)))
}

// FullXminHorizonStandby - Returns XminHorizonStandby in 64-bit FullTransactionId
func (ss PostgresServerStats) FullXminHorizonStandby() int64 {
	return int64(XidToXid8(Xid(ss.XminHorizonStandby.Int32), Xid8(ss.CurrentXactId)))
}
