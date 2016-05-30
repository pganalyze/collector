package state

import "gopkg.in/guregu/null.v3"

type PostgresRelation struct {
	Oid            Oid
	SchemaName     string
	RelationName   string
	RelationType   string
	Columns        []PostgresColumn
	Indices        []PostgresIndex
	Constraints    []PostgresConstraint
	ViewDefinition string
}

type PostgresColumn struct {
	RelationOid  Oid
	Name         string
	DataType     string
	DefaultValue null.String
	NotNull      bool
	Position     int32
}

type PostgresIndex struct {
	RelationOid   Oid
	IndexOid      Oid
	Columns       string
	Name          string
	IsPrimary     bool
	IsUnique      bool
	IsValid       bool
	IndexDef      string
	ConstraintDef null.String
}

type PostgresConstraint struct {
	RelationOid    Oid
	Name           string
	ConstraintDef  string
	Columns        null.String
	ForeignSchema  null.String
	ForeignTable   null.String
	ForeignColumns null.String
}
