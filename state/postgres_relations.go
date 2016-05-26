package state

import "gopkg.in/guregu/null.v3"

type PostgresRelation struct {
	Oid            Oid                   `json:"oid"`
	SchemaName     string                `json:"schema_name"`
	TableName      string                `json:"table_name"`
	RelationType   string                `json:"relation_type"`
	Stats          PostgresRelationStats `json:"stats"`
	Columns        []PostgresColumn      `json:"columns"`
	Indices        []PostgresIndex       `json:"indices"`
	Constraints    []PostgresConstraint  `json:"constraints"`
	ViewDefinition string                `json:"view_definition,omitempty"`
}

type PostgresRelationStats struct {
	SizeBytes        int64     `json:"size_bytes"`
	WastedBytes      int64     `json:"wasted_bytes"`
	SeqScan          null.Int  `json:"seq_scan"`            // Number of sequential scans initiated on this table
	SeqTupRead       null.Int  `json:"seq_tup_read"`        // Number of live rows fetched by sequential scans
	IdxScan          null.Int  `json:"idx_scan"`            // Number of index scans initiated on this table
	IdxTupFetch      null.Int  `json:"idx_tup_fetch"`       // Number of live rows fetched by index scans
	NTupIns          null.Int  `json:"n_tup_ins"`           // Number of rows inserted
	NTupUpd          null.Int  `json:"n_tup_upd"`           // Number of rows updated
	NTupDel          null.Int  `json:"n_tup_del"`           // Number of rows deleted
	NTupHotUpd       null.Int  `json:"n_tup_hot_upd"`       // Number of rows HOT updated (i.e., with no separate index update required)
	NLiveTup         null.Int  `json:"n_live_tup"`          // Estimated number of live rows
	NDeadTup         null.Int  `json:"n_dead_tup"`          // Estimated number of dead rows
	NModSinceAnalyze null.Int  `json:"n_mod_since_analyze"` // Estimated number of rows modified since this table was last analyzed
	LastVacuum       null.Time `json:"last_vacuum"`         // Last time at which this table was manually vacuumed (not counting VACUUM FULL)
	LastAutovacuum   null.Time `json:"last_autovacuum"`     // Last time at which this table was vacuumed by the autovacuum daemon
	LastAnalyze      null.Time `json:"last_analyze"`        // Last time at which this table was manually analyzed
	LastAutoanalyze  null.Time `json:"last_autoanalyze"`    // Last time at which this table was analyzed by the autovacuum daemon
	VacuumCount      null.Int  `json:"vacuum_count"`        // Number of times this table has been manually vacuumed (not counting VACUUM FULL)
	AutovacuumCount  null.Int  `json:"autovacuum_count"`    // Number of times this table has been vacuumed by the autovacuum daemon
	AnalyzeCount     null.Int  `json:"analyze_count"`       // Number of times this table has been manually analyzed
	AutoanalyzeCount null.Int  `json:"autoanalyze_count"`   // Number of times this table has been analyzed by the autovacuum daemon
	HeapBlksRead     null.Int  `json:"heap_blks_read"`      // Number of disk blocks read from this table
	HeapBlksHit      null.Int  `json:"heap_blks_hit"`       // Number of buffer hits in this table
	IdxBlksRead      null.Int  `json:"idx_blks_read"`       // Number of disk blocks read from all indexes on this table
	IdxBlksHit       null.Int  `json:"idx_blks_hit"`        // Number of buffer hits in all indexes on this table
	ToastBlksRead    null.Int  `json:"toast_blks_read"`     // Number of disk blocks read from this table's TOAST table (if any)
	ToastBlksHit     null.Int  `json:"toast_blks_hit"`      // Number of buffer hits in this table's TOAST table (if any)
	TidxBlksRead     null.Int  `json:"tidx_blks_read"`      // Number of disk blocks read from this table's TOAST table indexes (if any)
	TidxBlksHit      null.Int  `json:"tidx_blks_hit"`       // Number of buffer hits in this table's TOAST table indexes (if any)
}

type PostgresColumn struct {
	RelationOid  Oid         `json:"-"`
	Name         string      `json:"name"`
	DataType     string      `json:"data_type"`
	DefaultValue null.String `json:"default_value"`
	NotNull      bool        `json:"not_null"`
	Position     int32       `json:"position"`
}

type PostgresIndex struct {
	RelationOid   Oid         `json:"-"`
	IndexOid      Oid         `json:"-"`
	Columns       string      `json:"columns"`
	Name          string      `json:"name"`
	SizeBytes     int64       `json:"size_bytes"`
	WastedBytes   int64       `json:"wasted_bytes"`
	IsPrimary     bool        `json:"is_primary"`
	IsUnique      bool        `json:"is_unique"`
	IsValid       bool        `json:"is_valid"`
	IndexDef      string      `json:"index_def"`
	ConstraintDef null.String `json:"constraint_def"`
	IdxScan       null.Int    `json:"idx_scan"`
	IdxTupRead    null.Int    `json:"idx_tup_read"`
	IdxTupFetch   null.Int    `json:"idx_tup_fetch"`
	IdxBlksRead   null.Int    `json:"idx_blks_read"`
	IdxBlksHit    null.Int    `json:"idx_blks_hit"`
}

type PostgresConstraint struct {
	RelationOid    Oid         `json:"-"`
	Name           string      `json:"name"`
	ConstraintDef  string      `json:"constraint_def"`
	Columns        null.String `json:"columns"`
	ForeignSchema  null.String `json:"foreign_schema"`
	ForeignTable   null.String `json:"foreign_table"`
	ForeignColumns null.String `json:"foreign_columns"`
}
