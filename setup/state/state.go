package state

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-ini/ini"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/setup/log"
	"github.com/pganalyze/collector/setup/query"
	"github.com/pganalyze/collector/util"
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

	RecommendedInputs = SetupInputs{
		Scripted: true,

		Settings: RecommendedSettings,
		GUCS:     RecommendedGUCS,

		PGSetupConnPort: null.IntFrom(5432),
		PGSetupConnUser: null.StringFrom("postgres"),

		EnsureMonitoringUser:            null.BoolFrom(true),
		GenerateMonitoringPassword:      null.BoolFrom(true),
		EnsureMonitoringPassword:        null.BoolFrom(true),
		EnsureMonitoringPermissions:     null.BoolFrom(true),
		EnsureHelperFunctions:           null.BoolFrom(true),
		EnsurePgStatStatementsInstalled: null.BoolFrom(true),
		EnsurePgStatStatementsLoaded:    null.BoolFrom(true),

		GuessLogLocation: null.BoolFrom(true),

		UseLogBasedExplain:      null.BoolFrom(false),
		EnsureAutoExplainLoaded: null.BoolFrom(true),

		ConfirmPostgresRestart: null.BoolFrom(true),

		ConfirmSetUpLogInsights:              null.BoolFrom(true),
		ConfirmSetUpAutomatedExplain:         null.BoolFrom(true),
		EnsureAutoExplainRecommendedSettings: null.BoolFrom(true),
		ConfirmRunTestCommand:                null.BoolFrom(true),
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

	EnsureMonitoringUser            null.Bool `json:"ensure_monitoring_user"`
	GenerateMonitoringPassword      null.Bool `json:"generate_monitoring_password"`
	EnsureMonitoringPassword        null.Bool `json:"ensure_monitoring_password"`
	EnsureMonitoringPermissions     null.Bool `json:"ensure_monitoring_permissions"`
	EnsureHelperFunctions           null.Bool `json:"ensure_helper_functions"`
	EnsurePgStatStatementsInstalled null.Bool `json:"ensure_pg_stat_statements_installed"`
	EnsurePgStatStatementsLoaded    null.Bool `json:"ensure_pg_stat_statements_loaded"`

	GuessLogLocation null.Bool `json:"guess_log_location"`

	UseLogBasedExplain      null.Bool `json:"use_log_based_explain"`
	EnsureLogExplainHelpers null.Bool `json:"ensure_log_explain_helpers"`
	EnsureAutoExplainLoaded null.Bool `json:"ensure_auto_explain_loaded"`

	ConfirmPostgresRestart null.Bool `json:"confirm_postgres_restart"`

	ConfirmSetUpLogInsights              null.Bool `json:"confirm_set_up_log_insights"`
	ConfirmSetUpAutomatedExplain         null.Bool `json:"confirm_set_up_automated_explain"`
	EnsureAutoExplainRecommendedSettings null.Bool `json:"ensure_auto_explain_recommended_settings"`
	ConfirmRunTestCommand                null.Bool `json:"confirm_run_test_command"`
	ConfirmRunTestExplainCommand         null.Bool `json:"confirm_run_test_explain_command"`
}

var RecommendedInputs SetupInputs

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

	DidTestCommand                    bool
	DidTestExplainCommand             bool
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
	return state.Config.SaveTo(state.ConfigFilename)
}

func (state *SetupState) ReportStep(stepID string, stepErr error) {
	if !state.Inputs.Settings.APIKey.Valid || state.Inputs.Settings.APIKey.String == "" {
		return
	}
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second, DualStack: true}).DialContext(ctx, network, addr)
	}
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}

	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: transport,
	}

	var isSuccess string
	if stepErr == nil {
		isSuccess = "true"
	} else {
		isSuccess = "false"
	}
	var usedInputsFile string
	if state.Inputs.Scripted {
		usedInputsFile = "true"
	} else {
		usedInputsFile = "false"
	}
	data := url.Values{
		"last_step":        {stepID},
		"success":          {isSuccess},
		"used_inputs_file": {usedInputsFile},
	}
	var baseUrl = config.DefaultAPIBaseURL
	if state.Inputs.Settings.APIBaseURL.Valid {
		baseUrl = state.Inputs.Settings.APIBaseURL.String
	}
	reportURL := baseUrl + "/v2/setup/guided_setup"
	req, err := http.NewRequest("POST", reportURL, strings.NewReader(data.Encode()))
	if err != nil {
		// N.B. we don't care about this or any other errors--this is best-effort
		// reporting and should not affect the setup process
		return
	}

	req.Header.Set("Pganalyze-Api-Key", state.Inputs.Settings.APIKey.String)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	resp, err := client.Do(req)
	if err != nil {
		// again, ignoring errors
		return
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
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
	// Step ID
	ID string
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
