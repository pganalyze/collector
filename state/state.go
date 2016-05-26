package state

import (
	"database/sql"
	"time"

	"github.com/pganalyze/collector/config"
)

type State struct {
	CollectedAt time.Time

	Statements map[PostgresStatementKey]PostgresStatement

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
	Statements []DiffedPostgresStatement
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

type Database struct {
	Config           config.DatabaseConfig
	Connection       *sql.DB
	PrevState        State
	RequestedSslMode string
}
