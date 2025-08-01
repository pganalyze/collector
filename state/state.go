package state

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/gorilla/websocket"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/output/pganalyze_collector"
)

type SchemaStats struct {
	RelationStats         PostgresRelationStatsMap
	ColumnStats           PostgresColumnStatsMap
	RelationStatsExtended PostgresRelationStatsExtendedMap
	IndexStats            PostgresIndexStatsMap
	FunctionStats         PostgresFunctionStatsMap
}

// PersistedState - State thats kept across collector runs to be used for diffs
type PersistedState struct {
	CollectedAt time.Time

	DatabaseStats  PostgresDatabaseStatsMap
	StatementStats PostgresStatementStatsMap
	SchemaStats    map[Oid]*SchemaStats
	PlanStats      PostgresPlanStatsMap
	ServerIoStats  PostgresServerIoStatsMap

	Relations []PostgresRelation
	Functions []PostgresFunction

	System                SystemState
	CollectorStats        CollectorStats
	PgStatStatementsStats PgStatStatementsStats

	// Incremented every run, indicates whether we should run a pg_stat_statements_reset()
	// on behalf of the user. Only activates once it reaches GrantFeatures.StatementReset,
	// and is reset afterwards.
	StatementResetCounter int

	// Keep track of when we last collected statement stats, to calculate time distance
	LastStatementStatsAt time.Time

	// All statement stats that have not been identified (will be cleared by the next full snapshot)
	UnidentifiedStatementStats HistoricStatementStatsMap

	// All plan stats that have not been identified (will be cleared by the next full snapshot)
	UnidentifiedPlanStats HistoricPlanStatsMap

	// Data queued up for submission with the next full snapshot
	QueuedServerIoStats HistoricPostgresServerIoStatsMap
}

// TransientState - State thats only used within a collector run (and not needed for diffs)
type TransientState struct {
	// Databases we connected to and fetched local catalog data (e.g. schema)
	DatabaseOidsWithLocalCatalog []Oid

	Roles     []PostgresRole
	Databases []PostgresDatabase
	Types     []PostgresType

	Statements             PostgresStatementMap
	StatementTexts         PostgresStatementTextMap
	HistoricStatementStats HistoricStatementStatsMap
	HistoricPlanStats      HistoricPlanStatsMap
	Plans                  PostgresPlanMap
	HistoricServerIoStats  HistoricPostgresServerIoStatsMap

	// This is a new zero value that was recorded after a pg_stat_statements_reset(),
	// in order to enable the next snapshot to be able to diff against something
	ResetStatementStats PostgresStatementStatsMap

	BufferCache   BufferCache
	ServerStats   PostgresServerStats
	Replication   PostgresReplication
	Settings      []PostgresSetting
	BackendCounts []PostgresBackendCount
	Extensions    []PostgresExtension

	Version PostgresVersion

	SentryClient *raven.Client

	CollectorConfig   CollectorConfig
	CollectorPlatform CollectorPlatform
}

type BufferCache map[Oid]map[Oid]int64

type CollectorConfig struct {
	SectionName                string
	DisableLogs                bool
	DisableActivity            bool
	EnableLogExplain           bool
	DbName                     string
	DbUsername                 string
	DbHost                     string
	DbPort                     int32
	DbSslmode                  string
	DbHasSslrootcert           bool
	DbHasSslcert               bool
	DbHasSslkey                bool
	DbExtraNames               []string
	DbAllNames                 bool
	DbURLRedacted              string
	AwsRegion                  string
	AwsDbInstanceId            string
	AwsDbClusterID             string
	AwsDbClusterReadonly       bool
	AwsHasAccessKeyId          bool
	AwsHasAssumeRole           bool
	AwsHasAccountId            bool
	AwsHasWebIdentityTokenFile bool
	AwsHasRoleArn              bool
	AzureDbServerName          string
	AzureEventhubNamespace     string
	AzureEventhubName          string
	AzureAdTenantId            string
	AzureAdClientId            string
	AzureHasAdCertificate      bool
	AzureSubscriptionID        string
	GcpCloudsqlInstanceId      string
	GcpAlloyDBClusterID        string
	GcpAlloyDBInstanceID       string
	GcpPubsubSubscription      string
	GcpHasCredentialsFile      bool
	GcpProjectId               string
	GcpRegion                  string
	GcpUsePublicIP             bool
	CrunchyBridgeClusterId     string
	AivenProjectId             string
	AivenServiceId             string
	ApiSystemId                string
	ApiSystemType              string
	ApiSystemScope             string
	ApiSystemIdFallback        string
	ApiSystemTypeFallback      string
	ApiSystemScopeFallback     string
	DbLogLocation              string
	DbLogDockerTail            string
	DbLogSyslogServer          string
	DbLogPgReadFile            bool
	IgnoreTablePattern         string
	IgnoreSchemaRegexp         string
	QueryStatsInterval         int32
	MaxCollectorConnections    int32
	SkipIfReplica              bool
	MaxBufferCacheMonitoringGB int32
	FilterLogSecret            string
	FilterQuerySample          string
	FilterQueryText            string
	HasProxy                   bool
	ConfigFromEnv              bool
	OtelExporterOtlpEndpoint   string
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
	PlanStats      DiffedPostgresPlanStatsMap
	ServerIoStats  DiffedPostgresServerIoStatsMap

	SystemCPUStats     DiffedSystemCPUStatsMap
	SystemNetworkStats DiffedNetworkStatsMap
	SystemDiskStats    DiffedDiskStatsMap

	CollectorStats DiffedCollectorStats

	DatabaseStats DiffedPostgresDatabaseStatsMap

	PgStatStatementsStats DiffedPgStatStatementsStats
}

// StateOnDiskFormatVersion - Increment this when an old state preserved to disk should be ignored
const StateOnDiskFormatVersion = 6

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

	SubmitCollectedData              bool
	TestRun                          bool
	TestRunLogs                      bool
	TestExplain                      bool
	TestSection                      string
	GenerateStatsHelperSql           string
	GenerateExplainAnalyzeHelperSql  string
	GenerateExplainAnalyzeHelperRole string
	DebugLogs                        bool
	DiscoverLogLocation              bool

	StateFilename    string
	WriteStateUpdate bool
	ForceEmptyGrant  bool

	OutputAsJson bool
	VeryVerbose  bool
}

type Grant struct {
	Valid    bool
	Config   pganalyze_collector.ServerMessage_Config `json:"config"`
	S3URL    string                                   `json:"s3_url"`
	S3Fields map[string]string                        `json:"s3_fields"`
	LocalDir string                                   `json:"local_dir"`
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

type QueryRun struct {
	Id                  int64
	Type                pganalyze_collector.QueryRunType
	DatabaseName        string
	QueryText           string
	QueryParameters     []null.String
	QueryParameterTypes []string
	PostgresSettings    map[string]string
	Result              string
	Error               string
	StartedAt           time.Time
	FinishedAt          time.Time
	BackendPid          int
}

type Server struct {
	Config           config.ServerConfig
	RequestedSslMode string
	Grant            atomic.Pointer[Grant]

	PrevState  PersistedState
	StateMutex *sync.Mutex

	LogPrevState  PersistedLogState
	LogStateMutex *sync.Mutex

	ActivityPrevState  PersistedActivityState
	ActivityStateMutex *sync.Mutex

	CollectionStatus      CollectionStatus
	CollectionStatusMutex *sync.Mutex

	SelfTest *SelfTestResult

	SnapshotStream chan []byte
	WebSocket      atomic.Pointer[websocket.Conn]
	Pause          atomic.Bool

	// State to track queries the collector is running on behalf of a user
	QueryRuns      map[int64]*QueryRun
	QueryRunsMutex *sync.Mutex

	// The LogParser for this server, updated as necessary whenever relevant
	// settings (log_line_prefix and log_timezone) change
	// The LogSettingsMutex should be held while updating this
	LogParser     LogParser
	LogParseMutex *sync.RWMutex

	// Boolean flags for which log lines should be ignored for processing
	//
	// Internally this uses atomics (not a mutex) due to noticable performance
	// differences (see https://groups.google.com/g/golang-nuts/c/eIqkhXh9PLg),
	// as we access this in high frequency log-related code paths.
	LogIgnoreFlags uint32

	// State to track compact snapshot submissions, and log them routinely
	CompactLogStats map[string]uint8
	CompactLogTime  time.Time
}

func MakeServer(config config.ServerConfig, testRun bool) *Server {
	server := &Server{
		Config:                config,
		StateMutex:            &sync.Mutex{},
		LogStateMutex:         &sync.Mutex{},
		ActivityStateMutex:    &sync.Mutex{},
		CollectionStatusMutex: &sync.Mutex{},
		SnapshotStream:        make(chan []byte),
		QueryRuns:             make(map[int64]*QueryRun),
		QueryRunsMutex:        &sync.Mutex{},
		LogParseMutex:         &sync.RWMutex{},
	}
	server.Grant.Store(&Grant{Config: pganalyze_collector.ServerMessage_Config{Features: &pganalyze_collector.ServerMessage_Features{}}})
	server.Pause.Store(false)
	if testRun {
		server.SelfTest = MakeSelfTest()
	}
	return server
}

const (
	LOG_IGNORE_STATEMENT uint32 = 1 << iota
	LOG_IGNORE_DURATION
)

type LogParser interface {
	Matches(prefix string, tz *time.Location) bool
	GetOccurredAt(timePart string) time.Time
	ParseLine(line string) (logLine LogLine, ok bool)
	ValidatePrefix() error
	GetPrefixAndContent(line string) (prefix string, content string, ok bool)
}

func (s *Server) SetLogIgnoreFlags(ignoreStatement bool, ignoreDuration bool) {
	var newFlags uint32
	if ignoreStatement {
		newFlags |= LOG_IGNORE_STATEMENT
	}
	if ignoreDuration {
		newFlags |= LOG_IGNORE_DURATION
	}
	atomic.StoreUint32(&s.LogIgnoreFlags, newFlags)
}

func (s *Server) GetLogParser() LogParser {
	s.LogParseMutex.RLock()
	defer s.LogParseMutex.RUnlock()

	return s.LogParser
}

// IgnoreLogLine - helper function that lets callers determine whether a log
// line should be filtered out early (before any analysis)
//
// This is mainly intended to support Log Insights for servers that have very
// high log volume due to running with log_statement=all or log_duration=on
// (something we can't parse effectively with today's regexp-based log parsing),
// and allow other less frequent log events to be analyzed.
func (s *Server) IgnoreLogLine(content string) bool {
	flags := atomic.LoadUint32(&s.LogIgnoreFlags)

	return (flags&LOG_IGNORE_STATEMENT != 0 && (strings.HasPrefix(content, "statement: ") || strings.HasPrefix(content, "execute ") || strings.HasPrefix(content, "parameters: "))) ||
		(flags&LOG_IGNORE_DURATION != 0 && strings.HasPrefix(content, "duration: ") && !strings.Contains(content, " ms  plan:\n"))
}
