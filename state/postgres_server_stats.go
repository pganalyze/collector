package state

// PostgresServerStats - Statistics for a Postgres server.
type PostgresServerStats struct {
	CurrentXactId   Xid
	NextMultiXactId Xid
}
