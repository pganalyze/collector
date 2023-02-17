package state

import "github.com/guregu/null"

type Oid uint64
type Xid uint32  // 32-bit transaction ID
type Xid8 uint64 // 64-bit transaction ID

// PostgresType - User-defined custom data types
type PostgresType struct {
	Oid               Oid
	ArrayOid          Oid
	DatabaseOid       Oid
	SchemaName        string
	Name              string
	Type              string
	DomainType        null.String
	DomainNotNull     bool
	DomainDefault     null.String
	DomainConstraints []string
	EnumValues        []string
	CompositeAttrs    [][2]string
}

// XidToXid8 - Converts Xid (32-bit transaction ID) to Xid8 (64-bit FullTransactionId)
// by calculating and adding an epoch from the current transaction ID
func XidToXid8(xid Xid, currentXactId Xid8) Xid8 {
	// Do not proceed the conversion if either of the input is 0
	// The currentXactID can be 0 on replicas
	if xid == 0 || currentXactId == 0 {
		return 0
	}
	// If we simply shift the currentXactId, it'll give the epoch of the current transaction ID, which may be different
	// from the epoch of the given xid (the one we want to add).
	// By subtracting the xid from the current one, we can get the epoch of the given xid.
	xidEpoch := int32((currentXactId - Xid8(xid)) >> 32)
	return Xid8(xidEpoch)<<32 | Xid8(xid)
}
