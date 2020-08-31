package state

import (
	"sync"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/pganalyze/collector/config"
)

type DBStats struct {
	RelationStats PostgresRelationStatsMap
	IndexStats    PostgresIndexStatsMap
	FunctionStats PostgresFunctionStatsMap
}

// PersistedState - State thats kept across collector runs to be used for diffs
type PersistedState struct {
	CollectedAt time.Time

	StatementStats PostgresStatementStatsMap
	DBStats        map[Oid]*DBStats

	Relations []PostgresRelation
	Functions []PostgresFunction

	System         SystemState
	CollectorStats CollectorStats

	// Incremented every run, indicates whether we should run a pg_stat_statements_reset()
	// on behalf of the user. Only activates once it reaches GrantFeatures.StatementReset,
	// and is reset afterwards.
	StatementResetCounter int

	// Keep track of when we last collected statement stats, to calculate time distance
	LastStatementStatsAt time.Time

	// All statement stats that have not been identified (will be cleared by the next full snapshot)
	UnidentifiedStatementStats HistoricStatementStatsMap
}

// TransientState - State thats only used within a collector run (and not needed for diffs)
type TransientState struct {
	// Databases we connected to and fetched local catalog data (e.g. schema)
	DatabaseOidsWithLocalCatalog []Oid

	Roles     []PostgresRole
	Databases []PostgresDatabase

	Statements             PostgresStatementMap
	StatementTexts         PostgresStatementTextMap
	HistoricStatementStats HistoricStatementStatsMap

	// This is a new zero value that was recorded after a pg_stat_statements_reset(),
	// in order to enable the next snapshot to be able to diff against something
	ResetStatementStats PostgresStatementStatsMap

	Replication   PostgresReplication
	Settings      []PostgresSetting
	BackendCounts []PostgresBackendCount

	Version PostgresVersion

	SentryClient *raven.Client
}

type DiffedDBStats struct {
	RelationStats DiffedPostgresRelationStatsMap
	IndexStats    DiffedPostgresIndexStatsMap
	FunctionStats DiffedPostgresFunctionStatsMap
}

// DiffState - Result of diff-ing two persistent state structs
type DiffState struct {
	StatementStats DiffedPostgresStatementStatsMap
	DBStats        map[Oid]*DiffedDBStats

	SystemCPUStats     DiffedSystemCPUStatsMap
	SystemNetworkStats DiffedNetworkStatsMap
	SystemDiskStats    DiffedDiskStatsMap

	CollectorStats DiffedCollectorStats
}

// StateOnDiskFormatVersion - Increment this when an old state preserved to disk should be ignored
const StateOnDiskFormatVersion = 3

type StateOnDisk struct {
	FormatVersion uint

	PrevStateByServer map[config.ServerIdentifier]PersistedState
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
	TestReport          string
	TestRunLogs         bool
	DebugLogs           bool
	DiscoverLogLocation bool

	StateFilename    string
	WriteStateUpdate bool
	ForceEmptyGrant  bool
}

type GrantConfig struct {
	ServerID  string `json:"server_id"`
	SentryDsn string `json:"sentry_dsn"`

	Features GrantFeatures `json:"features"`

	EnableActivity bool `json:"enable_activity"`
	EnableLogs     bool `json:"enable_logs"`
}

type GrantFeatures struct {
	Logs bool `json:"logs"`

	StatementResetFrequency     int   `json:"statement_reset_frequency"`
	StatementTimeoutMs          int32 `json:"statement_timeout_ms"`            // Statement timeout for all SQL statements sent to the database (defaults to 30s)
	StatementTimeoutMsQueryText int32 `json:"statement_timeout_ms_query_text"` // Statement timeout for pg_stat_statements query text requests (defaults to 120s)
}

type Grant struct {
	Valid    bool
	Config   GrantConfig       `json:"config"`
	S3URL    string            `json:"s3_url"`
	S3Fields map[string]string `json:"s3_fields"`
	LocalDir string            `json:"local_dir"`
}

func (g Grant) S3() GrantS3 {
	return GrantS3{S3URL: g.S3URL, S3Fields: g.S3Fields}
}

type GrantS3 struct {
	S3URL    string            `json:"s3_url"`
	S3Fields map[string]string `json:"s3_fields"`
}

type Server struct {
	Config           config.ServerConfig
	RequestedSslMode string
	Grant            Grant

	PrevState  PersistedState
	StateMutex *sync.Mutex

	LogPrevState  PersistedLogState
	LogStateMutex *sync.Mutex

	ActivityPrevState  PersistedActivityState
	ActivityStateMutex *sync.Mutex
}
