package state

import (
	"sort"

	"github.com/go-ini/ini"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/setup/log"
	"github.com/pganalyze/collector/setup/query"
)

var SupportedLogLinePrefixes []string

func init() {
	SupportedLogLinePrefixes = make([]string, len(logs.SupportedPrefixes))
	copy(SupportedLogLinePrefixes, logs.SupportedPrefixes)
	recommended := SupportedLogLinePrefixes[logs.RecommendedPrefixIdx]

	sort.SliceStable(SupportedLogLinePrefixes, func(i, j int) bool {
		if SupportedLogLinePrefixes[i] == recommended {
			return true
		} else if SupportedLogLinePrefixes[j] == recommended {
			return false
		} else {
			return i < j
		}
	})

	RecommendedGUCS = SetupGUCS{
		LogErrorVerbosity:       null.StringFrom("default"),
		LogDuration:             null.StringFrom("off"),
		LogStatement:            null.StringFrom("none"),
		LogMinDurationStatement: null.IntFrom(1000),
		LogLinePrefix:           null.StringFrom(SupportedLogLinePrefixes[0]),

		AutoExplainLogAnalyze:          null.StringFrom("on"),
		AutoExplainLogBuffers:          null.StringFrom("on"),
		AutoExplainLogTiming:           null.StringFrom("off"),
		AutoExplainLogTriggers:         null.StringFrom("on"),
		AutoExplainLogVerbose:          null.StringFrom("on"),
		AutoExplainLogFormat:           null.StringFrom("json"),
		AutoExplainLogMinDuration:      null.IntFrom(1000),
		AutoExplainLogNestedStatements: null.StringFrom("on"),
	}
}

type SetupSettings struct {
	APIKey        null.String `json:"api_key"`
	APIBaseURL    null.String `json:"api_base_url"`
	DBName        null.String `json:"db_name"`
	DBUsername    null.String `json:"db_username"`
	DBPassword    null.String `json:"db_password"`
	DBLogLocation null.String `json:"db_log_location"`
}

var RecommendedSettings = SetupSettings{
	DBUsername: null.StringFrom("pganalyze"),
}

type SetupGUCS struct {
	LogErrorVerbosity       null.String `json:"log_error_verbosity"`
	LogDuration             null.String `json:"log_duration"`
	LogStatement            null.String `json:"log_statement"`
	LogMinDurationStatement null.Int    `json:"log_min_duration_statement"`
	LogLinePrefix           null.String `json:"log_line_prefix"`

	AutoExplainLogAnalyze          null.String `json:"auto_explain.log_analyze"`
	AutoExplainLogBuffers          null.String `json:"auto_explain.log_buffers"`
	AutoExplainLogTiming           null.String `json:"auto_explain.log_timing"`
	AutoExplainLogTriggers         null.String `json:"auto_explain.log_triggers"`
	AutoExplainLogVerbose          null.String `json:"auto_explain.log_verbose"`
	AutoExplainLogFormat           null.String `json:"auto_explain.log_format"`
	AutoExplainLogMinDuration      null.Int    `json:"auto_explain.log_min_duration"`
	AutoExplainLogNestedStatements null.String `json:"auto_explain.log_nested_statements"`
}

var RecommendedGUCS SetupGUCS

type SetupInputs struct {
	Scripted bool

	Settings SetupSettings `json:"settings"`
	GUCS     SetupGUCS     `json:"gucs"`

	PGSetupConnSocketDir null.String `json:"pg_setup_conn_socket_dir"`
	PGSetupConnPort      null.Int    `json:"pg_setup_conn_port"`
	PGSetupConnUser      null.String `json:"pg_setup_conn_user"`

	CreateMonitoringUser       null.Bool `json:"create_monitoring_user"`
	GenerateMonitoringPassword null.Bool `json:"generate_monitoring_password"`
	UpdateMonitoringPassword   null.Bool `json:"update_monitoring_password"`
	SetUpMonitoringUser        null.Bool `json:"set_up_monitoring_user"`
	CreateHelperFunctions      null.Bool `json:"create_helper_functions"`
	CreatePgStatStatements     null.Bool `json:"create_pg_stat_statements"`
	EnablePgStatStatements     null.Bool `json:"enable_pg_stat_statements"`

	GuessLogLocation null.Bool `json:"guess_log_location"`

	UseLogBasedExplain  null.Bool `json:"use_log_based_explain"`
	CreateExplainHelper null.Bool `json:"create_explain_helper"`
	EnableAutoExplain   null.Bool `json:"enable_auto_explain"`

	ConfirmCollectorReload null.Bool `json:"confirm_collector_reload"`
	ConfirmPostgresRestart null.Bool `json:"confirm_postgres_restart"`

	SkipLogInsights            null.Bool `json:"skip_log_insights"`
	SkipAutomatedExplain       null.Bool `json:"skip_automated_explain"`
	SkipAutoExplainRecommended null.Bool `json:"skip_automated_explain_recommended_settings"`
	SkipPgSleep                null.Bool `json:"skip_pg_sleep"`
}

var RecommendedInputs = SetupInputs{
	Scripted: true,

	Settings: RecommendedSettings,
	GUCS:     RecommendedGUCS,

	PGSetupConnPort: null.IntFrom(5432),
	PGSetupConnUser: null.StringFrom("postgres"),

	CreateMonitoringUser:       null.BoolFrom(true),
	GenerateMonitoringPassword: null.BoolFrom(true),
	UpdateMonitoringPassword:   null.BoolFrom(true),
	SetUpMonitoringUser:        null.BoolFrom(true),
	CreateHelperFunctions:      null.BoolFrom(true),
	CreatePgStatStatements:     null.BoolFrom(true),
	EnablePgStatStatements:     null.BoolFrom(true),

	GuessLogLocation: null.BoolFrom(true),

	UseLogBasedExplain: null.BoolFrom(false),
	EnableAutoExplain:  null.BoolFrom(true),

	ConfirmCollectorReload: null.BoolFrom(true),
	ConfirmPostgresRestart: null.BoolFrom(true),

	SkipLogInsights:            null.BoolFrom(false),
	SkipAutomatedExplain:       null.BoolFrom(false),
	SkipAutoExplainRecommended: null.BoolFrom(false),
	SkipPgSleep:                null.BoolFrom(false),
}

type SetupState struct {
	OperatingSystem string
	Platform        string
	PlatformFamily  string
	PlatformVersion string

	QueryRunner  *query.Runner
	PGVersionNum int
	PGVersionStr string

	ConfigFilename   string
	Config           *ini.File
	CurrentSection   *ini.Section
	PGAnalyzeSection *ini.Section

	Inputs *SetupInputs

	NeedsReload bool

	DidReload                         bool
	DidPgSleep                        bool
	DidAutoExplainRecommendedSettings bool

	Logger *log.Logger
}

func (state *SetupState) Log(line string, params ...interface{}) error {
	return state.Logger.Log(line, params...)
}

func (state *SetupState) Verbose(line string, params ...interface{}) error {
	return state.Logger.Verbose(line, params...)
}

func (state *SetupState) SaveConfig() error {
	state.NeedsReload = true
	return state.Config.SaveTo(state.ConfigFilename)
}

type StepKind int

const (
	GeneralStep          StepKind = 0
	LogInsightsStep      StepKind = 1
	AutomatedExplainStep StepKind = 2
)

// Step is a discrete step in the install process
type Step struct {
	// Kind of step
	Kind StepKind
	// Description of what the step entails
	Description string
	// Check if the step has already been completed--may modify the state struct, but
	// never modifies Postgres, the collector config, or anything else in the installed
	// system
	Check func(state *SetupState) (bool, error)
	// Make changes to the system necessary for the check to pass, always prompting for
	// user input before any change that modifies Postgres, the collector config, or
	// anything else in the installed system
	Run func(state *SetupState) error
}
