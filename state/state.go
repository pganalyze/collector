package state

import (
	"sync"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/pganalyze/collector/config"
)

type SchemaStats struct {
	RelationStats PostgresRelationStatsMap
	IndexStats    PostgresIndexStatsMap
	FunctionStats PostgresFunctionStatsMap
}

// PersistedState - State thats kept across collector runs to be used for diffs
type PersistedState struct {
	CollectedAt time.Time

	StatementStats PostgresStatementStatsMap
	SchemaStats    map[Oid]*SchemaStats

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

	CollectorConfig   CollectorConfig
	CollectorPlatform CollectorPlatform
}

type CollectorConfig struct {
	SectionName             string
	DisableLogs             bool
	DisableActivity         bool
	EnableLogExplain        bool
	DbName                  string
	DbUsername              string
	DbHost                  string
	DbPort                  int32
	DbSslmode               string
	DbHasSslrootcert        bool
	DbHasSslcert            bool
	DbHasSslkey             bool
	DbExtraNames            []string
	DbAllNames              bool
	DbURLRedacted           string
	AwsRegion               string
	AwsDbInstanceId         string
	AwsHasAccessKeyId       bool
	AwsHasAssumeRole        bool
	AwsHasAccountId         bool
	AzureDbServerName       string
	AzureEventhubNamespace  string
	AzureEventhubName       string
	AzureAdTenantId         string
	AzureAdClientId         string
	AzureHasAdCertificate   bool
	GcpCloudsqlInstanceId   string
	GcpPubsubSubscription   string
	GcpHasCredentialsFile   bool
	GcpProjectId            string
	ApiSystemId             string
	ApiSystemType           string
	ApiSystemScope          string
	ApiSystemScopeFallback  string
	DbLogLocation           string
	DbLogDockerTail         string
	IgnoreTablePattern      string
	IgnoreSchemaRegexp      string
	QueryStatsInterval      int32
	MaxCollectorConnections int32
	SkipIfReplica           bool
	FilterLogSecret         string
	FilterQuerySample       string
	FilterQueryText         string
	HasProxy                bool
	ConfigFromEnv           bool
}

type CollectorPlatform struct {
	StartedAt            time.Time
	Architecture         string
	Hostname             string
	OperatingSystem      string
	Platform             string
	PlatformFamily       string
	PlatformVersion      string
	VirtualizationSystem string
	KernelVersion        string
}

type DiffedSchemaStats struct {
	RelationStats DiffedPostgresRelationStatsMap
	IndexStats    DiffedPostgresIndexStatsMap
	FunctionStats DiffedPostgresFunctionStatsMap
}

// DiffState - Result of diff-ing two persistent state structs
type DiffState struct {
	StatementStats DiffedPostgresStatementStatsMap
	SchemaStats    map[Oid]*DiffedSchemaStats

	SystemCPUStats     DiffedSystemCPUStatsMap
	SystemNetworkStats DiffedNetworkStatsMap
	SystemDiskStats    DiffedDiskStatsMap

	CollectorStats DiffedCollectorStats
}

// StateOnDiskFormatVersion - Increment this when an old state preserved to disk should be ignored
const StateOnDiskFormatVersion = 5

type StateOnDisk struct {
	FormatVersion uint

	PrevStateByServer map[config.ServerIdentifier]PersistedState
}

type CollectionOpts struct {
	StartedAt time.Time

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
	TestExplain         bool
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

type CollectionStatus struct {
	CollectionDisabled        bool
	CollectionDisabledReason  string
	LogSnapshotDisabled       bool
	LogSnapshotDisabledReason string
}

type Server struct {
	Config           config.ServerConfig
	RequestedSslMode string
	Grant            Grant
	PGAnalyzeURL     string

	PrevState  PersistedState
	StateMutex *sync.Mutex

	LogPrevState  PersistedLogState
	LogStateMutex *sync.Mutex

	ActivityPrevState  PersistedActivityState
	ActivityStateMutex *sync.Mutex

	CollectionStatus      CollectionStatus
	CollectionStatusMutex *sync.Mutex
}
