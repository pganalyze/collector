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

	Roles     []PostgresRole
	Databases []PostgresDatabase
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

// StateOnDiskFormatVersion - Increment this when an old state preserved to disk should be ignored
const StateOnDiskFormatVersion = 1

type StateOnDisk struct {
	FormatVersion uint

	PrevStateByAPIKey map[string]State
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

	StateFilename string
}

type GrantConfig struct {
	// Here be dragons
}

type Grant struct {
	Valid    bool
	Config   GrantConfig       `json:"config"`
	S3URL    string            `json:"s3_url"`
	S3Fields map[string]string `json:"s3_fields"`
}

type Server struct {
	Config           config.ServerConfig
	Connection       *sql.DB
	PrevState        State
	RequestedSslMode string
	Grant            Grant
}
