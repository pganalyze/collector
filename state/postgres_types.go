package state

type Oid uint64
type Xid uint32

// PostgresType - User-defined custom data types
type PostgresType struct {
	Oid             Oid
	DatabaseOid     Oid
	SchemaName      string
	Name            string
	Type            string
	UnderlyingType  string
	NotNull         bool
	Default         string
	Constraint      string
	EnumValues      []string
	CompositeAttrs  [][2]string
}
