//go:generate msgp

package snapshot

type Oid int64

type Relation struct {
	Oid            Oid           `msg:"oid"`
	SchemaName     string        `msg:"schema_name"`
	TableName      string        `msg:"table_name"`
	RelationType   string        `msg:"relation_type"`
	Stats          RelationStats `msg:"stats"`
	Columns        []Column      `msg:"columns"`
	Indices        []Index       `msg:"indices"`
	Constraints    []Constraint  `msg:"constraints"`
	ViewDefinition string        `msg:"view_definition,omitempty"`
}

type RelationStats struct {
	SizeBytes        int64                 `msg:"size_bytes"`
	WastedBytes      int64                 `msg:"wasted_bytes"`
	SeqScan          NullableInt           `msg:"seq_scan"`            // Number of sequential scans initiated on this table
	SeqTupRead       NullableInt           `msg:"seq_tup_read"`        // Number of live rows fetched by sequential scans
	IdxScan          NullableInt           `msg:"idx_scan"`            // Number of index scans initiated on this table
	IdxTupFetch      NullableInt           `msg:"idx_tup_fetch"`       // Number of live rows fetched by index scans
	NTupIns          NullableInt           `msg:"n_tup_ins"`           // Number of rows inserted
	NTupUpd          NullableInt           `msg:"n_tup_upd"`           // Number of rows updated
	NTupDel          NullableInt           `msg:"n_tup_del"`           // Number of rows deleted
	NTupHotUpd       NullableInt           `msg:"n_tup_hot_upd"`       // Number of rows HOT updated (i.e., with no separate index update required)
	NLiveTup         NullableInt           `msg:"n_live_tup"`          // Estimated number of live rows
	NDeadTup         NullableInt           `msg:"n_dead_tup"`          // Estimated number of dead rows
	NModSinceAnalyze NullableInt           `msg:"n_mod_since_analyze"` // Estimated number of rows modified since this table was last analyzed
	LastVacuum       NullableUnixTimestamp `msg:"last_vacuum"`         // Last time at which this table was manually vacuumed (not counting VACUUM FULL)
	LastAutovacuum   NullableUnixTimestamp `msg:"last_autovacuum"`     // Last time at which this table was vacuumed by the autovacuum daemon
	LastAnalyze      NullableUnixTimestamp `msg:"last_analyze"`        // Last time at which this table was manually analyzed
	LastAutoanalyze  NullableUnixTimestamp `msg:"last_autoanalyze"`    // Last time at which this table was analyzed by the autovacuum daemon
	VacuumCount      NullableInt           `msg:"vacuum_count"`        // Number of times this table has been manually vacuumed (not counting VACUUM FULL)
	AutovacuumCount  NullableInt           `msg:"autovacuum_count"`    // Number of times this table has been vacuumed by the autovacuum daemon
	AnalyzeCount     NullableInt           `msg:"analyze_count"`       // Number of times this table has been manually analyzed
	AutoanalyzeCount NullableInt           `msg:"autoanalyze_count"`   // Number of times this table has been analyzed by the autovacuum daemon
	HeapBlksRead     NullableInt           `msg:"heap_blks_read"`      // Number of disk blocks read from this table
	HeapBlksHit      NullableInt           `msg:"heap_blks_hit"`       // Number of buffer hits in this table
	IdxBlksRead      NullableInt           `msg:"idx_blks_read"`       // Number of disk blocks read from all indexes on this table
	IdxBlksHit       NullableInt           `msg:"idx_blks_hit"`        // Number of buffer hits in all indexes on this table
	ToastBlksRead    NullableInt           `msg:"toast_blks_read"`     // Number of disk blocks read from this table's TOAST table (if any)
	ToastBlksHit     NullableInt           `msg:"toast_blks_hit"`      // Number of buffer hits in this table's TOAST table (if any)
	TidxBlksRead     NullableInt           `msg:"tidx_blks_read"`      // Number of disk blocks read from this table's TOAST table indexes (if any)
	TidxBlksHit      NullableInt           `msg:"tidx_blks_hit"`       // Number of buffer hits in this table's TOAST table indexes (if any)
}

type Column struct {
	RelationOid  Oid            `msg:"-"`
	Name         string         `msg:"name"`
	DataType     string         `msg:"data_type"`
	DefaultValue NullableString `msg:"default_value"`
	NotNull      bool           `msg:"not_null"`
	Position     int32          `msg:"position"`
}

type Index struct {
	RelationOid   Oid            `msg:"-"`
	IndexOid      Oid            `msg:"-"`
	Columns       string         `msg:"columns"`
	Name          string         `msg:"name"`
	SizeBytes     int64          `msg:"size_bytes"`
	WastedBytes   int64          `msg:"wasted_bytes"`
	IsPrimary     bool           `msg:"is_primary"`
	IsUnique      bool           `msg:"is_unique"`
	IsValid       bool           `msg:"is_valid"`
	IndexDef      string         `msg:"index_def"`
	ConstraintDef NullableString `msg:"constraint_def"`
	IdxScan       NullableInt    `msg:"idx_scan"`
	IdxTupRead    NullableInt    `msg:"idx_tup_read"`
	IdxTupFetch   NullableInt    `msg:"idx_tup_fetch"`
	IdxBlksRead   NullableInt    `msg:"idx_blks_read"`
	IdxBlksHit    NullableInt    `msg:"idx_blks_hit"`
}

type Constraint struct {
	RelationOid    Oid            `msg:"-"`
	Name           string         `msg:"name"`
	ConstraintDef  string         `msg:"constraint_def"`
	Columns        NullableString `msg:"columns"`
	ForeignSchema  NullableString `msg:"foreign_schema"`
	ForeignTable   NullableString `msg:"foreign_table"`
	ForeignColumns NullableString `msg:"foreign_columns"`
}
