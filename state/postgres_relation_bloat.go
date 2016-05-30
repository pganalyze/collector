package state

type PostgresRelationBloat struct {
	WastedBytes int64
	SizeBytes   int64
}

type PostgresIndexBloat struct {
	WastedBytes int64
	SizeBytes   int64
}

type PostgresRelationBloatMap map[Oid]PostgresRelationBloat
type PostgresIndexBloatMap map[Oid]PostgresIndexBloat
