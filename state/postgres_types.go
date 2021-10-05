package state

import "github.com/guregu/null"

type Oid uint64
type Xid uint32

// PostgresType - User-defined custom data types
type PostgresType struct {
	Oid              Oid
	DatabaseOid      Oid
	SchemaName       string
	Name             string
	Type             string
	DomainType       null.String
	DomainNotNull    bool
	DomainDefault    null.String
	DomainConstraint null.String
	EnumValues       []string
	CompositeAttrs   [][2]string
}
