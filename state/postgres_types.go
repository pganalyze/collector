package state

import "github.com/guregu/null"

type Oid uint64
type Xid uint32

// PostgresType - User-defined custom data types
type PostgresType struct {
	Oid             Oid
	DatabaseOid     Oid
	SchemaName      string
	Name            string
	Type            string
	UnderlyingType  null.String
	NotNull         bool
	Default         null.String
	Constraint      null.String
	EnumValues      []string
	CompositeAttrs  [][2]string
}
