//go:generate msgp

package snapshot

type Snapshot struct {
	ActiveQueries []Activity       `msg:"backends"`
	Statements    []Statement      `msg:"queries"`
	Postgres      SnapshotPostgres `msg:"postgres"`
	System        *System          `msg:"system"`
	Logs          []LogLine        `msg:"logs"`
	Explains      []Explain        `msg:"explains"`
	Opts          SnapshotOpts     `msg:"opts"`
}

type SnapshotOpts struct {
	StatementStatsAreDiffed        bool `msg:"statement_stats_are_diffed"`
	PostgresRelationStatsAreDiffed bool `msg:"postgres_relation_stats_are_diffed"`
}

type SnapshotPostgres struct {
	Relations []Relation      `msg:"schema"`
	Settings  []Setting       `msg:"settings"`
	Functions []Function      `msg:"functions"`
	Version   PostgresVersion `msg:"version"`
}
