package state

// PostgresDatabase - A database in the PostgreSQL system, with multiple schemas and tables contained in it
type PostgresDatabase struct {
	Oid              Oid    // ID of this database
	Name             string // Database name
	OwnerRoleOid     Oid    // Owner of the database, usually the user who created it
	Encoding         string // Character encoding for this database
	Collate          string // LC_COLLATE for this database
	CType            string // LC_CTYPE for this database
	IsTemplate       bool   // If true, then this database can be cloned by any user with CREATEDB privileges; if false, then only superusers or the owner of the database can clone it.
	AllowConnections bool   // If false then no one can connect to this database. This is used to protect the template0 database from being altered.
	ConnectionLimit  int32  // Sets maximum number of concurrent connections that can be made to this database. -1 means no limit.

	// All transaction IDs before this one have been replaced with a permanent ("frozen") transaction ID in this database.
	// This is used to track whether the database needs to be vacuumed in order to prevent transaction ID wraparound or to
	// allow pg_clog to be shrunk. It is the minimum of the per-table pg_class.relfrozenxid values.
	FrozenXID Xid

	// All multixact IDs before this one have been replaced with a transaction ID in this database.
	// This is used to track whether the database needs to be vacuumed in order to prevent multixact ID wraparound or to
	// allow pg_multixact to be shrunk. It is the minimum of the per-table pg_class.relminmxid values.
	MinimumMultixactXID Xid

	XIDAge  uint32 // Age of FrozenXID
	MXIDAge uint32 // Age of MinimumMultixactXID
}

// PostgresDatabaseStats - Database statistics for a single database
type PostgresDatabaseStats struct {
	TransactionCount uint64 // Transaction count of this database
}

// PostgresDatabaseStatsMap - Map of database statistics (key = database Oid)
type PostgresDatabaseStatsMap map[Oid]PostgresDatabaseStats

// DiffedPostgresDatabaseStat - Database statistics for a single database as a diff
type DiffedPostgresDatabaseStats struct {
	TransactionsPerSecond uint64
}

// DiffedDatabaseStats - Map of diffed database statistics (key = database Oid)
type DiffedPostgresDatabaseStatsMap map[Oid]DiffedPostgresDatabaseStats

// DiffSince - Calculate the diff between two stats runs
func (curr PostgresDatabaseStats) DiffSince(prev PostgresDatabaseStats, collectedIntervalSecs uint32) DiffedPostgresDatabaseStats {
	return DiffedPostgresDatabaseStats{
		TransactionsPerSecond: (curr.TransactionCount - prev.TransactionCount) / uint64(collectedIntervalSecs),
	}
}
