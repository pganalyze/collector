package state

import (
	"database/sql"
	"time"

	"github.com/pganalyze/collector/config"
)

type State struct {
	CollectedAt time.Time

	Statements    PostgresStatementMap
	RelationStats PostgresRelationStatsMap
	IndexStats    PostgresIndexStatsMap

	Backends  []PostgresBackend
	Relations []PostgresRelation
	Settings  []PostgresSetting
	Functions []PostgresFunction
	Version   PostgresVersion
	System    *SystemState
	Logs      []LogLine
	Explains  []PostgresExplain
}

type DiffState struct {
	Statements    []DiffedPostgresStatement
	RelationStats DiffedPostgresRelationStatsMap
	IndexStats    DiffedPostgresIndexStatsMap
}

type CollectionOpts struct {
	CollectPostgresRelations bool
	CollectPostgresSettings  bool
	CollectPostgresLocks     bool
	CollectPostgresFunctions bool
	CollectPostgresBloat     bool
	CollectPostgresViews     bool

	CollectLogs              bool
	CollectExplain           bool
	CollectSystemInformation bool

	CollectorApplicationName string

	DiffStatements bool

	SubmitCollectedData bool
	TestRun             bool
}

type GrantConfig struct {
	// Here be dragons
}

type Grant struct {
	Config   GrantConfig       `json:"config"`
	S3URL    string            `json:"s3_url"`
	S3Fields map[string]string `json:"s3_fields"`
}

type Database struct {
	Config           config.DatabaseConfig
	Connection       *sql.DB
	PrevState        State
	RequestedSslMode string
	Grant            Grant
}
