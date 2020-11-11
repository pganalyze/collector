package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/go-ini/ini"
	"github.com/lib/pq"
	"github.com/shirou/gopsutil/host"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/setup/query"
	"github.com/pganalyze/collector/setup/service"
	"github.com/pganalyze/collector/util"
)

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

	AskedLogInsights      bool
	SkipLogInsights       bool
	AskedAutomatedExplain bool
	SkipAutomatedExplain  bool

	LogBasedExplain bool

	DidReload bool
}

// Step is a discrete step in the install process
type Step struct {
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

const defaultConfigFile = "/etc/pganalyze-collector.conf"

func main() {
	steps := []*Step{
		determinePlatform,
		loadConfig,
		saveAPIKey,
		establishSuperuserConnection,
		checkPostgresVersion,
		checkReplicationStatus,
		selectDatabases,
		specifyMonitoringUser,
		createMonitoringUser,
		configureMonitoringUserPasswd,
		applyMonitoringUserPasswd,
		setUpMonitoringUser,
		checkPgssAvailable,
		createPgss,
		enablePgss,

		confirmLogInsightsSetup,
		configureLogErrorVerbosity,
		configureLogDuration,
		configureLogStatement,
		configureLogMinDurationStatement,
		configureLogLinePrefix,
		configureLogLocation,

		confirmAutoExplainSetup,
		checkAutoExplainAvailable,
		enableAutoExplain,
		configureAutoExplain,

		reloadCollector,
		restartPg,
	}

	var setupState SetupState
	flag.StringVar(&setupState.ConfigFilename, "config", defaultConfigFile, "Specify alternative path for config file")
	flag.Parse()

	// TODO: check for root?

	fmt.Println(`Welcome to the pganalyze collector installer!

We will go through a series of steps to set up the collector to monitor your
Postgres database. We will not make any changes to your database or system
without confirmation.

At a high level, we will:

 1. Configure a database user and helper functions for the collector, with minimal access
 2. Update the collector configuration file with these settings
 3. Set up the pg_stat_statements extension in your database for basic query performance monitoring
 4. (Optional) Make log-related configuration settings changes to enable our Log Insights feature
 5. (Optional) Set up EXPLAIN plan collection to enable our Automated EXPLAIN feature

At each step, we'll check if any changes are necessary, and if so, prompt you to
provide input or confirm any required changes.

You can stop at any time by pressing Ctrl+C.

If you stop before completing setup, you can resume by running the installer
again. We can pick up where you left off.`)
	fmt.Println()
	var doSetup bool
	err := survey.AskOne(&survey.Confirm{
		Message: "Continue with setup?",
		Default: false,
	}, &doSetup)
	if err != nil {
		fmt.Printf("  automated setup failed: %s\n", err)
	}
	if !doSetup {
		fmt.Println("Exiting...")
		os.Exit(0)
	}

	for _, step := range steps {
		err := doStep(&setupState, step)
		if err != nil {
			fmt.Printf("    automated setup failed: %s\n", err)
			return
		}
	}
}

func doStep(setupState *SetupState, step *Step) error {
	if step.Check == nil {
		panic("step missing completion check")
	}
	fmt.Printf("%s %s: ", bold("*"), step.Description)
	done, err := step.Check(setupState)
	if err != nil {
		fmt.Println("✗")
		return err
	}
	if done {
		fmt.Println("✓")
		return nil
	}
	if step.Run == nil {
		// panic because we should always define a Run func if a check can fail
		panic("check failed and no resolution defined")
	}
	fmt.Println("?")

	err = step.Run(setupState)
	if err != nil {
		return err
	}

	fmt.Print("    re-checking: ")
	done, err = step.Check(setupState)
	if err != nil {
		fmt.Println("✗")
		return err
	}
	if !done {
		return errors.New("check still failed after running resolution; please try again")
	}
	fmt.Println("✓")
	return nil
}

var determinePlatform = &Step{
	Description: "Determine platform",
	Check: func(state *SetupState) (bool, error) {
		hostInfo, err := host.Info()
		if err != nil {
			return false, err
		}
		state.OperatingSystem = hostInfo.OS
		state.Platform = hostInfo.Platform
		state.PlatformFamily = hostInfo.PlatformFamily
		state.PlatformVersion = hostInfo.PlatformVersion

		// TODO: relax this
		if state.Platform != "ubuntu" || state.PlatformVersion != "20.04" {
			return false, errors.New("automated setup only supported on Ubuntu 20.04")
		}

		return true, nil
	},
}

var loadConfig = &Step{
	Description: "Load collector config",
	Check: func(state *SetupState) (bool, error) {
		config, err := ini.Load(state.ConfigFilename)
		if err != nil {
			return false, err
		}

		// TODO: relax this
		if len(config.Sections()) != 3 {
			// N.B.: DEFAULT section, pganalyze section, server section
			return false, fmt.Errorf("not supported for config file defining more than one server")
		}
		state.Config = config
		for _, section := range config.Sections() {
			if section.Name() == "pganalyze" {
				state.PGAnalyzeSection = section
			} else if section.Name() == "DEFAULT" {
				continue
			} else {
				state.CurrentSection = section
			}
		}

		if state.CurrentSection.HasKey("db_url") {
			return false, errors.New("not supported when db_url is already configured")
		}

		return true, nil
	},
}

var saveAPIKey = &Step{
	Description: "Add pganalyze API key to collector config",
	Check: func(state *SetupState) (bool, error) {
		return state.PGAnalyzeSection.HasKey("api_key"), nil
	},
	Run: func(state *SetupState) error {
		apiKey := os.Getenv("PGA_API_KEY")
		if apiKey == "" {
			err := survey.AskOne(&survey.Input{
				Message: "PGA_API_KEY environment variable not found; please enter API key (will be saved to collector config):",
				Help:    "The key can be found on the API keys page for your organization in the pganalyze app",
			}, &apiKey, survey.WithValidator(survey.Required))
			if err != nil {
				return err
			}
		} else {
			var configWriteConfirmed bool
			err := survey.AskOne(&survey.Confirm{
				Message: "PGA_API_KEY found in environment; save to config file?",
				Default: false,
			}, &configWriteConfirmed)
			if err != nil {
				return err
			}
			if !configWriteConfirmed {
				return nil
			}
		}
		_, err := state.PGAnalyzeSection.NewKey("api_key", apiKey)
		if err != nil {
			return err
		}
		return state.Config.SaveTo(state.ConfigFilename)
	},
}

var establishSuperuserConnection = &Step{
	Description: "Ensure Postgres superuser connection",
	Check: func(state *SetupState) (bool, error) {
		if state.QueryRunner == nil {
			return false, nil
		}
		err := state.QueryRunner.Ping()
		return err == nil, err
	},
	Run: func(state *SetupState) error {
		// TODO: Ping and prompt for credentials if ping fails; even if ping succeed, confirm
		// the connection we are using
		state.QueryRunner = query.NewRunner()
		return nil
	},
}

var checkPostgresVersion = &Step{
	Description: "Check Postgres version",
	Check: func(state *SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow("SELECT current_setting('server_version'), current_setting('server_version_num')::integer")
		if err != nil {
			return false, err
		}
		state.PGVersionStr = row.GetString(0)
		state.PGVersionNum = row.GetInt(1)

		return true, nil
	},
}

var checkReplicationStatus = &Step{
	Description: "Check replication status",
	Check: func(state *SetupState) (bool, error) {
		result, err := state.QueryRunner.QueryRow("SELECT pg_is_in_recovery()")
		if err != nil {
			return false, err
		}
		isReplicationTarget := result.GetBool(0)

		if isReplicationTarget {
			return false, errors.New("not supported for replicas")
		}
		return true, nil
	},
}

var selectDatabases = &Step{
	Description: "Select database(s) to monitor",
	Check: func(state *SetupState) (bool, error) {
		key, err := state.CurrentSection.GetKey("db_name")
		if err != nil {
			return false, err
		}
		db := key.String()
		if db == "" {
			return false, nil
		}
		// Now that we know the database, connect to the right one for setup:
		// this is important for extensions and helper functions
		state.QueryRunner.Database = db
		return true, nil
	},
	Run: func(state *SetupState) error {
		rows, err := state.QueryRunner.Query("SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate")
		if err != nil {
			return err
		}
		var dbOpts []string
		for _, row := range rows {
			dbOpts = append(dbOpts, row.GetString(0))
		}

		// TODO: monitor other databases or *
		var selectedDb string
		err = survey.AskOne(&survey.Select{
			Message: "Choose a primary database to monitor (will be saved to collector config):",
			Options: dbOpts,
		}, &selectedDb)
		if err != nil {
			return err
		}

		_, err = state.CurrentSection.NewKey("db_name", selectedDb)
		if err != nil {
			return err
		}

		return state.Config.SaveTo(state.ConfigFilename)
	},
}

var specifyMonitoringUser = &Step{
	Description: "Check config for monitoring user",
	Check: func(state *SetupState) (bool, error) {
		hasUser := state.CurrentSection.HasKey("db_username")
		return hasUser, nil
	},
	Run: func(state *SetupState) error {
		var monitoringUser int
		err := survey.AskOne(&survey.Select{
			Message: "Select Postgres user for the collector to use (will be saved to collector config):",
			Help:    "If the user does not exist, it can be created in a later step",
			Options: []string{"pganalyze (recommended)", "a different user"},
		}, &monitoringUser)
		if err != nil {
			return err
		}
		var pgaUser string
		if monitoringUser == 0 {
			pgaUser = "pganalyze"
		} else if monitoringUser == 1 {
			err := survey.AskOne(&survey.Input{
				Message: "Enter Postgres user for the collector to use (will be saved to collector config):",
				Help:    "If the user does not exist, it can be created in a later step",
			}, &pgaUser, survey.WithValidator(survey.Required))
			if err != nil {
				return err
			}
		} else {
			panic(fmt.Sprintf("unexpected user selection: %d", monitoringUser))
		}

		_, err = state.CurrentSection.NewKey("db_username", pgaUser)
		if err != nil {
			return err
		}
		return state.Config.SaveTo(state.ConfigFilename)
	},
}

var createMonitoringUser = &Step{
	Description: "Ensure monitoring user exists",
	Check: func(state *SetupState) (bool, error) {
		pgaUserKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return false, err
		}
		pgaUser := pgaUserKey.String()

		var result query.Row
		result, err = state.QueryRunner.QueryRow(fmt.Sprintf("SELECT true FROM pg_user WHERE usename = %s", pq.QuoteLiteral(pgaUser)))
		if err == query.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return result.GetBool(0), nil
	},
	Run: func(state *SetupState) error {
		pgaUserKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()

		var doCreateUser bool
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("User %s does not exist in Postgres; create user (will be saved to Postgres)?", pgaUser),
			Help:    "If you skip this step, create the user manually before proceeding",
			Default: false,
		}, &doCreateUser)
		if err != nil {
			return err
		}
		if !doCreateUser {
			return nil
		}

		return state.QueryRunner.Exec(
			fmt.Sprintf(
				"CREATE USER %s",
				pq.QuoteIdentifier(pgaUser),
			),
		)
	},
}

var configureMonitoringUserPasswd = &Step{
	Description: "Configure monitoring user password",
	Check: func(state *SetupState) (bool, error) {
		hasPassword := state.CurrentSection.HasKey("db_password")
		return hasPassword, nil
	},
	Run: func(state *SetupState) error {
		var passwordStrategy int
		err := survey.AskOne(&survey.Select{
			Message: "Select how to set up the collector user password (will be saved to collector config):",
			Options: []string{"generate random password (recommended)", "enter existing password"},
		}, &passwordStrategy)
		if err != nil {
			return err
		}

		var pgaPasswd string
		if passwordStrategy == 0 {
			passwdBytes := make([]byte, 16)
			rand.Read(passwdBytes)
			pgaPasswd = hex.EncodeToString(passwdBytes)
		} else if passwordStrategy == 1 {
			err = survey.AskOne(&survey.Input{
				Message: "Enter password for the collector to use (will be saved to collector config):",
			}, &passwordStrategy, survey.WithValidator(survey.Required))
			if err != nil {
				return err
			}
		}
		_, err = state.CurrentSection.NewKey("db_password", pgaPasswd)
		if err != nil {
			return err
		}

		return state.Config.SaveTo(state.ConfigFilename)
	},
}

var applyMonitoringUserPasswd = &Step{
	Description: "Apply monitoring user password",
	Check: func(state *SetupState) (bool, error) {
		cfg, err := config.Read(
			&util.Logger{Destination: log.New(os.Stderr, "", 0)},
			state.ConfigFilename,
		)
		if err != nil {
			return false, err
		}
		if len(cfg.Servers) != 1 {
			return false, fmt.Errorf("expected one server in config; found %d", len(cfg.Servers))
		}
		serverCfg := cfg.Servers[0]
		pqStr := serverCfg.GetPqOpenString("")
		conn, err := sql.Open("postgres", pqStr)
		err = conn.Ping()
		isAuthErr := strings.Contains(err.Error(), "authentication failed")
		if isAuthErr {
			return false, nil
		}
		return false, err
	},
	Run: func(state *SetupState) error {
		pgaUserKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()
		pgaPasswdKey, err := state.CurrentSection.GetKey("db_password")
		if err != nil {
			return err
		}
		pgaPasswd := pgaPasswdKey.String()

		var doPasswdUpdate bool
		err = survey.AskOne(&survey.Confirm{
			Message: "Update password for user %s with configured value (will be saved to Postgres)?",
			Help:    "If you skip this step, ensure the password matches before proceeding",
		}, &doPasswdUpdate)
		if err != nil {
			return err
		}
		if !doPasswdUpdate {
			return nil
		}
		err = state.QueryRunner.Exec(
			fmt.Sprintf(
				"ALTER USER %s WITH ENCRYPTED PASSWORD %s",
				pq.QuoteIdentifier(pgaUser),
				pq.QuoteLiteral(pgaPasswd),
			),
		)
		return err
	},
}

// ensure user has correct permissions
var setUpMonitoringUser = &Step{
	Description: "Set up monitoring user",
	Check: func(state *SetupState) (bool, error) {
		pgaUserKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return false, err
		}
		pgaUser := pgaUserKey.String()

		// TODO: deal with postgres <10
		// TODO: more lenient check--we don't necessarily need role membership--just access to
		// all the necessary tables and functions
		row, err := state.QueryRunner.QueryRow(
			fmt.Sprintf(
				"SELECT usesuper OR pg_has_role(usename, 'pg_monitor', 'member') FROM pg_user WHERE usename = %s",
				pq.QuoteLiteral(pgaUser),
			),
		)
		if err == query.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}

		return row.GetBool(0), nil
	},
	Run: func(state *SetupState) error {
		pgaUserKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()

		// TODO: deal with postgres <10
		var doGrant bool
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Grant role pg_monitor to user %s (will be saved to Postgres)?", pgaUser),
			Help:    "Learn more about pg_monitor here: https://www.postgresql.org/docs/current/default-roles.html",
		}, &doGrant)
		if err != nil {
			return err
		}
		if !doGrant {
			return nil
		}

		return state.QueryRunner.Exec(
			fmt.Sprintf(
				"GRANT pg_monitor to %s",
				pq.QuoteIdentifier(pgaUser),
			),
		)
	},
}

var checkPgssAvailable = &Step{
	Description: "Prepare for pg_stat_statements install",
	Check: func(state *SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(
			fmt.Sprintf(
				"SELECT true FROM pg_available_extensions WHERE name = 'pg_stat_statements'",
			),
		)
		if err == query.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return row.GetBool(0), nil
	},
	Run: func(state *SetupState) error {
		// TODO: install contrib package?
		return errors.New("extension pg_stat_statements is not available")
	},
}

// install pg_stat_statements extension
var createPgss = &Step{
	Description: "Install pg_stat_statements",
	Check: func(state *SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(
			fmt.Sprintf(
				"SELECT extnamespace::regnamespace::text FROM pg_extension WHERE extname = 'pg_stat_statements'",
			),
		)
		if err == query.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}
		extNsp := row.GetString(0)
		if extNsp != "public" {
			return false, fmt.Errorf("pg_stat_statements is installed, but in unsupported schema %s; must be installed in 'public'", extNsp)
		}
		return true, nil
	},
	Run: func(state *SetupState) error {
		var doCreate bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Create extension pg_stat_statements in public schema (will be saved to Postgres)?",
			Default: false,
			Help:    "Learn more about pg_stat_statements here: https://www.postgresql.org/docs/current/pgstatstatements.html",
		}, &doCreate)
		if err != nil {
			return err
		}
		if !doCreate {
			return nil
		}
		return state.QueryRunner.Exec("CREATE EXTENSION pg_stat_statements SCHEMA public")
	},
}

var enablePgss = &Step{
	Description: "Enable pg_stat_statements",
	Check: func(state *SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow("SHOW shared_preload_libraries")
		if err != nil {
			return false, err
		}
		return strings.Contains(row.GetString(0), "pg_stat_statements"), nil
	},
	Run: func(state *SetupState) error {
		var doAdd bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Add pg_stat_statements to shared_preload_libraries (will be saved to Postgres)?",
			Default: false,
			Help:    "Postgres will have to be restarted in a later step to apply this configuration change; learn more about shared_preload_libraries here: https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SHARED-PRELOAD-LIBRARIES",
		}, &doAdd)
		if err != nil {
			return err
		}
		if !doAdd {
			return nil
		}

		row, err := state.QueryRunner.QueryRow("SHOW shared_preload_libraries")
		if err != nil {
			return err
		}
		var existingSpl = row.GetString(0)
		var newSpl string
		if existingSpl == "" {
			newSpl = "pg_stat_statements"
		} else {
			newSpl = existingSpl + ",pg_stat_statements"
		}
		return applyConfigSetting("shared_preload_libraries", newSpl, state.QueryRunner)
	},
}

var confirmLogInsightsSetup = &Step{
	Description: "Check whether Log Insights should be configured",
	Check: func(state *SetupState) (bool, error) {
		return state.AskedLogInsights, nil
	},
	Run: func(state *SetupState) error {
		var setUpLogInsights bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Proceed to configuring optional Log Insights feature?",
			Help:    "Learn more at https://pganalyze.com/log-insights",
			Default: false,
		}, &setUpLogInsights)
		if err != nil {
			return err
		}
		state.AskedLogInsights = true
		state.SkipLogInsights = !setUpLogInsights

		return nil
	},
}

var configureLogErrorVerbosity = &Step{
	Description: "Check log_error_verbosity",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipLogInsights {
			return true, nil
		}
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_error_verbosity'`)
		if err != nil {
			return false, err
		}

		return row.GetString(0) != "verbose", nil
	},
	Run: func(state *SetupState) error {
		var newVal string
		err := survey.AskOne(&survey.Select{
			Message: "Setting 'log_error_verbosity' is set to unsupported value 'all'; select supported value (will be saved to Postgres):",
			Options: []string{"terse", "default"},
		}, &newVal)
		if err != nil {
			return err
		}
		return applyConfigSetting("log_error_verbosity", newVal, state.QueryRunner)
	},
}

var configureLogDuration = &Step{
	Description: "Check log_duration",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipLogInsights {
			return true, nil
		}
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_duration'`)
		if err != nil {
			return false, err
		}

		return row.GetString(0) == "off", nil
	},
	Run: func(state *SetupState) error {
		var turnOffLogDuration bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Setting 'log_duration' is set to unsupported value 'on'; set to 'off' (will be saved to Postgres)?",
			Default: false,
		}, &turnOffLogDuration)
		if err != nil {
			return err
		}
		if !turnOffLogDuration {
			// technically there is no error to report here; the re-check will fail
			return nil
		}
		return applyConfigSetting("log_duration", "off", state.QueryRunner)
	},
}

var configureLogStatement = &Step{
	Description: "Check log_statement",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipLogInsights {
			return true, nil
		}
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_statement'`)
		if err != nil {
			return false, err
		}

		return row.GetString(0) != "all", nil
	},
	Run: func(state *SetupState) error {
		var newVal string
		err := survey.AskOne(&survey.Select{
			Message: "Setting 'log_statement' is set to unsupported value 'all'; select supported value (will be saved to Postgres):",
			Options: []string{"none", "ddl", "mod"},
		}, &newVal)
		if err != nil {
			return err
		}

		return applyConfigSetting("log_statement", newVal, state.QueryRunner)
	},
}

func validateLmds(ans interface{}) error {
	ansStr, ok := ans.(string)
	if !ok {
		return errors.New("expected string value")
	}
	ansNum, err := strconv.Atoi(ansStr)
	if err != nil {
		return errors.New("value must be numeric")
	}
	if ansNum < 10 && ansNum != 0 {
		return errors.New("value must be either 0 or 10 or greater")
	}
	return nil
}

var configureLogMinDurationStatement = &Step{
	Description: "Check log_min_duration_statement",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipLogInsights {
			return true, nil
		}
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_min_duration_statement'`)
		if err != nil {
			return false, err
		}

		lmdsVal := row.GetInt(0)
		return lmdsVal == -1 || lmdsVal >= 10, nil
	},
	Run: func(state *SetupState) error {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_min_duration_statement'`)
		if err != nil {
			return err
		}
		oldVal := fmt.Sprintf("%sms", row.GetString(0))

		var newVal string
		err = survey.AskOne(&survey.Input{
			Message: fmt.Sprintf(
				"Setting 'log_min_duration_statement' is set to '%s', below supported threshold of 10ms; enter supported value in ms or 0 to disable (will be saved to Postgres):",
				oldVal,
			),
		}, &newVal, survey.WithValidator(validateLmds))
		if err != nil {
			return err
		}

		return applyConfigSetting("log_min_duration_statement", newVal, state.QueryRunner)
	},
}

var supportedLogLinePrefixes = []string{
	"%m [%p] %q[user=%u,db=%d,app=%a] ",
	"%m [%p] %q[user=%u,db=%d,app=%a,host=%h] ",
	"%t:%r:%u@%d:[%p]:",
	"%t [%p-%l] %q%u@%d ",
	"%t [%p]: [%l-1] user=%u,db=%d - PG-%e ",
	"%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h ",
	"%t [%p]: [%l-1] [trx_id=%x] user=%u,db=%d ",
	"%m %r %u %a [%c] [%p] ",
	"%m [%p][%v] : [%l-1] %q[app=%a] ",
	"%m [%p] ",
}

var configureLogLinePrefix = &Step{
	Description: "Check log_line_prefix",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipLogInsights {
			return true, nil
		}
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_line_prefix'`)
		if err != nil {
			return false, err
		}

		return includes(supportedLogLinePrefixes, row.GetString(0)), nil
	},
	Run: func(state *SetupState) error {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_line_prefix'`)
		if err != nil {
			return err
		}
		oldVal := row.GetString(0)
		var opts []string
		for i, llp := range supportedLogLinePrefixes {
			// N.B.: we quote the options because many prefixes end in whitespace; we need to make that clear
			var opt string
			if i == 0 {
				opt = fmt.Sprintf("'%s' (recommended)", llp)
			} else {
				opt = fmt.Sprintf("'%s'", llp)
			}
			opts = append(opts, opt)
		}

		var prefixIdx int
		err = survey.AskOne(&survey.Select{
			Message: fmt.Sprintf("Setting 'log_line_prefix' is set to unsupported value '%s'; set to (will be saved to Postgres):", oldVal),
			Help:    "Check format specifier reference in Postgres documentation: https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-LINE-PREFIX",
			Options: opts,
		}, &prefixIdx, survey.WithPageSize(len(supportedLogLinePrefixes)))
		if err != nil {
			return err
		}
		selectedPrefix := supportedLogLinePrefixes[prefixIdx]
		return applyConfigSetting("log_line_prefix", selectedPrefix, state.QueryRunner)
	},
}

func validatePath(ans interface{}) error {
	ansStr, ok := ans.(string)
	if !ok {
		return errors.New("expected string value")
	}
	// TODO: also confirm this is readable by the regular pganalyze user
	_, err := os.Stat(ansStr)
	if err != nil {
		return err
	}

	return nil
}

var configureLogLocation = &Step{
	Description: "Determine log location",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipLogInsights {
			return true, nil
		}
		return state.CurrentSection.HasKey("db_log_location"), nil
	},
	Run: func(state *SetupState) error {
		guessedLogLocation, err := discoverLogLocation(state.CurrentSection, state.QueryRunner)
		if err != nil {
			return err
		}
		var logLocationConfirmed bool
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Your database log file or directory appears to be %s; is this correct (will be saved to collector config)?", guessedLogLocation),
			Default: false,
		}, &logLocationConfirmed)
		if err != nil {
			return err
		}
		var logLocation string
		if logLocationConfirmed {
			logLocation = guessedLogLocation
		} else {
			err = survey.AskOne(&survey.Input{
				Message: "Please enter the Postgres log file location (will be saved to collector config)",
			}, &logLocation, survey.WithValidator(validatePath))
		}
		_, err = state.CurrentSection.NewKey("db_log_location", logLocation)
		if err != nil {
			return err
		}
		return state.Config.SaveTo(state.ConfigFilename)
	},
}

var confirmAutoExplainSetup = &Step{
	Description: "Check whether to configure Automated EXPLAIN",
	Check: func(state *SetupState) (bool, error) {
		return state.AskedAutomatedExplain, nil
	},
	Run: func(state *SetupState) error {
		var setUpExplain bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Proceed to configuring optional Automated EXPLAIN feature?",
			Help:    "Learn more at https://pganalyze.com/postgres-explain",
			Default: false,
		}, &setUpExplain)
		if err != nil {
			return err
		}
		state.AskedAutomatedExplain = true
		state.SkipAutomatedExplain = !setUpExplain

		return nil
	},
}

var checkAutoExplainAvailable = &Step{
	Description: "Prepare for auto_explain install",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipAutomatedExplain {
			return true, nil
		}
		err := state.QueryRunner.Exec("LOAD auto_explain")
		if err != nil {
			if strings.Contains(err.Error(), "No such file or directory") {
				return false, nil
			}

			return false, err
		}
		return true, err
	},
	Run: func(state *SetupState) error {
		// TODO: install contrib package?
		return errors.New("module auto_explain is not available")
	},
}

var enableAutoExplain = &Step{
	Description: "Enable auto_explain",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipAutomatedExplain {
			return true, nil
		}
		row, err := state.QueryRunner.QueryRow("SHOW shared_preload_libraries")
		if err != nil {
			return false, err
		}
		return strings.Contains(row.GetString(0), "auto_explain"), nil
	},
	Run: func(state *SetupState) error {
		var doAdd bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Add auto_explain to shared_preload_libraries (will be saved to Postgres)?",
			Default: false,
			Help:    "Postgres will have to be restarted in a later step to apply this configuration change; learn more about shared_preload_libraries here: https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SHARED-PRELOAD-LIBRARIES",
		}, &doAdd)
		if err != nil {
			return err
		}
		if !doAdd {
			return nil
		}
		row, err := state.QueryRunner.QueryRow("SHOW shared_preload_libraries")
		if err != nil {
			return err
		}
		var existingSpl = row.GetString(0)
		var newSpl string
		if existingSpl == "" {
			newSpl = "auto_explain"
		} else {
			newSpl = existingSpl + ",auto_explain"
		}
		return applyConfigSetting("shared_preload_libraries", newSpl, state.QueryRunner)
	},
}

var configureAutoExplain = &Step{
	Description: "Configure auto_explain",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipAutomatedExplain {
			return true, nil
		}
		// TODO: check recommended/required configuration settings
		return false, nil
	},
	Run: func(state *SetupState) error {
		return nil
	},
}

var reloadCollector = &Step{
	Description: "Reload collector configuration",
	Check: func(state *SetupState) (bool, error) {
		return state.DidReload, nil
	},
	Run: func(state *SetupState) error {
		err := reload()
		if err != nil {
			return err
		}
		state.DidReload = true
		return nil
	},
}

var restartPg = &Step{
	Description: "If necessary, restart Postgres to have configuration changes take effect",
	Check: func(state *SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow("SELECT COUNT(*) FROM pg_settings WHERE pending_restart;")
		if err != nil {
			return false, err
		}
		return row.GetInt(0) == 0, nil
	},
	Run: func(state *SetupState) error {
		rows, err := state.QueryRunner.Query("SELECT name FROM pg_settings WHERE pending_restart")
		if err != nil {
			return err
		}
		var pendingSettings []string
		for _, row := range rows {
			pendingSettings = append(pendingSettings, row.GetString(0))
		}

		pendingList := getConjuctionList(pendingSettings)
		var restartNow bool
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Postgres must be restarted for changes to %s to take effect; restart Postgres now?", pendingList),
			Default: false,
		}, &restartNow)
		if err != nil {
			return err
		}

		if !restartNow {
			return nil
		}

		return service.RestartPostgres()
	},
}
