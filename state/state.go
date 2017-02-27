package state

import (
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/pganalyze/collector/config"
)

// PersistedState - State thats kept across collector runs to be used for diffs
type PersistedState struct {
	CollectedAt time.Time

	StatementStats PostgresStatementStatsMap
	RelationStats  PostgresRelationStatsMap
	IndexStats     PostgresIndexStatsMap
	FunctionStats  PostgresFunctionStatsMap

	Relations []PostgresRelation
	Functions []PostgresFunction

	System         SystemState
	CollectorStats CollectorStats
}

// TransientState - State thats only used within a collector run (and not needed for diffs)
type TransientState struct {
	// Databases we connected to and fetched local catalog data (e.g. schema)
	DatabaseOidsWithLocalCatalog []Oid

	Roles     []PostgresRole
	Databases []PostgresDatabase

	Statements PostgresStatementMap

	Replication PostgresReplication
	Backends    []PostgresBackend
	Logs        []LogLine
	Explains    []PostgresExplain
	Settings    []PostgresSetting

	Version PostgresVersion

	SentryClient *raven.Client
}

// DiffState - Result of diff-ing two persistent state structs
type DiffState struct {
	StatementStats DiffedPostgresStatementStatsMap
	RelationStats  DiffedPostgresRelationStatsMap
	IndexStats     DiffedPostgresIndexStatsMap
	FunctionStats  DiffedPostgresFunctionStatsMap

	SystemCPUStats     DiffedSystemCPUStatsMap
	SystemNetworkStats DiffedNetworkStatsMap
	SystemDiskStats    DiffedDiskStatsMap

	CollectorStats DiffedCollectorStats
}

// StateOnDiskFormatVersion - Increment this when an old state preserved to disk should be ignored
const StateOnDiskFormatVersion = 1

type StateOnDisk struct {
	FormatVersion uint

	PrevStateByAPIKey map[string]PersistedState
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
	StatementTimeoutMs       int32 // Statement timeout for all SQL statements sent to the database

	DiffStatements bool

	SubmitCollectedData bool
	TestRun             bool
	TestReport          string

	StateFilename    string
	WriteStateUpdate bool
}

type GrantConfig struct {
	ServerID  string `json:"server_id"`
	SentryDsn string `json:"sentry_dsn"`
}

type Grant struct {
	Valid    bool
	Config   GrantConfig       `json:"config"`
	S3URL    string            `json:"s3_url"`
	S3Fields map[string]string `json:"s3_fields"`
	LocalDir string            `json:"local_dir"`
}

type Server struct {
	Config           config.ServerConfig
	PrevState        PersistedState
	RequestedSslMode string
	Grant            Grant
}
