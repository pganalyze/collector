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
}
