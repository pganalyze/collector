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
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/go-ini/ini"
	"github.com/lib/pq"
	"github.com/shirou/gopsutil/host"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/setup/query"
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

	UseLogInsights bool
	UseAutoExplain bool
	UseLogExplain  bool
}

type Step struct {
	Description string
	// check if the step has already been completed--may modify state
	Check func(state *SetupState) (bool, error)
	// apply the step, possibly with user input--note that some steps that
	// can be done automatically may not have a Run func
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
		restartPg,
		configureLogSettings,
		checkAutoExplainAvailable,
		enableAutoExplain,
		configureAutoExplain,
		restartPg, // N.B.: this is in this list twice
	}

	var setupState SetupState
	flag.StringVar(&setupState.ConfigFilename, "config", defaultConfigFile, "Specify alternative path for config file")
	flag.Parse()

	// TODO: check for root?

	fmt.Println(`Welcome to the pganalyze collector installer

We will go through a series of steps to set up the collector to monitor your
Postgres database. At each step, we'll check if any changes are necessary,
and if so, prompt you to provide input or just proceed to make the change.

You can stop at any time by pressing Ctrl+C.

If you stop before completing setup, you can resume by running the installer
again. We can pick up where you left off.
	`)
	var doSetup bool
	survey.AskOne(&survey.Confirm{
		Message: "Continue with setup?",
		Default: false,
	}, &doSetup)
	if !doSetup {
		fmt.Println("exiting...")
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
	fmt.Printf(" * %s: ", step.Description)
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
		return errors.New("check still failed after running resolution")
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
			survey.AskOne(&survey.Input{
				Message: "PGA_API_KEY environment variable not found; please enter API key:",
				Help:    "The key can be found on the API keys page for your organization in the pganalyze app",
			}, &apiKey, survey.WithValidator(survey.Required))
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
		// TODO: Ping and prompt for credentials if ping fails
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
			return false, errors.New("not supported for replication target")
		}
		return true, nil
	},
}

var selectDatabases = &Step{
	Description: "Select database(s) to monitor",
	Check: func(state *SetupState) (bool, error) {
		return state.CurrentSection.HasKey("db_name"), nil
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

		var selectedDb string
		survey.AskOne(&survey.Select{
			Message: "Choose a primary database to monitor:",
			Options: dbOpts,
		}, &selectedDb)

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
		survey.AskOne(&survey.Select{
			Message: "Select Postgres user for the collector to use:",
			Help:    "If the user does not exist, it can be created in a later step",
			Options: []string{"pganalyze (recommended)", "a different user"},
		}, &monitoringUser)
		var pgaUser string
		if monitoringUser == 0 {
			pgaUser = "pganalyze"
		} else if monitoringUser == 1 {
			survey.AskOne(&survey.Input{
				Message: "Enter Postgres user for the collector to use:",
				Help:    "If the user does not exist, it can be created in a later step",
			}, &pgaUser, survey.WithValidator(survey.Required))
		} else {
			panic(fmt.Sprintf("unexpected selection: %d", monitoringUser))
		}

		_, err := state.CurrentSection.NewKey("db_username", pgaUser)
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

		err = state.QueryRunner.Exec(
			fmt.Sprintf(
				"CREATE USER %s",
				pq.QuoteIdentifier(pgaUser),
			),
		)
		return err
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
			Message: "Select how to set up the collector user password:",
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
				Message: "Enter password for the collector to use:",
			}, &passwordStrategy)
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
		return err == nil, err
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
			Message: "Update password for user %s in Postgres with configured value?",
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
				"SELECT true FROM pg_user WHERE usename = %s AND (usesuper OR pg_has_role(usename, 'pg_monitor', 'member'))",
				pq.QuoteLiteral(pgaUser),
			),
		)
		if err == query.ErrNoRows {
			fmt.Printf("nope")
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
		err = state.QueryRunner.Exec(
			fmt.Sprintf(
				"GRANT pg_monitor to %s",
				pq.QuoteIdentifier(pgaUser),
			),
		)
		return err
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
		return nil
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
		return state.QueryRunner.Exec(
			fmt.Sprintf("ALTER SYSTEM SET shared_preload_libraries = %s", pq.QuoteLiteral(newSpl)),
		)
	},
}

var restartPg = &Step{
	Description: "Restart Postgres to have configuration changes take effect",
	Check: func(state *SetupState) (bool, error) {
		rows, err := state.QueryRunner.Query("SELECT name FROM pg_settings WHERE pending_restart;")
		if err != nil {
			return false, err
		}
		if len(rows) > 0 {
			return false, nil
		}
		return true, nil
	},
	Run: func(state *SetupState) error {
		// Postgres must be restarted for changes to the following settings to take effect:
		// "select name from pg_settings where pending_restart;"
		// Note that if you also intend to install auto_explain, that will also require a
		// restart--you can skip restarting now (check whether one of the settings is shared_preload_libraries and whether its value includes auto_explain)
		// allow this to set a SkipRestart flag that's checked and cleared in Check
		return nil
	},
}

// log_error_verbosity, log_duration, log_min_duration_statement, log_statement, log_line_prefix
var configureLogSettings = &Step{
	Description: "Configure logging-related settings",
	Check: func(state *SetupState) (bool, error) {
		// TODO: check recommended/required configuration settings
		return false, nil
	},
	Run: func(state *SetupState) error {
		return nil
	},
}

var checkAutoExplainAvailable = &Step{
	Description: "Prepare for auto_explain install",
	Check: func(state *SetupState) (bool, error) {
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
		return nil
	},
}

var enableAutoExplain = &Step{
	Description: "Enable auto_explain",
	Check: func(state *SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow("SHOW shared_preload_libraries")
		if err != nil {
			return false, err
		}
		return strings.Contains(row.GetString(0), "auto_explain"), nil
	},
	Run: func(state *SetupState) error {
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
		return state.QueryRunner.Exec(
			fmt.Sprintf("ALTER SYSTEM SET shared_preload_libraries = %s", pq.QuoteLiteral(newSpl)),
		)
	},
}

var configureAutoExplain = &Step{
	Description: "Configure auto_explain",
	Check: func(state *SetupState) (bool, error) {
		// TODO: check recommended/required configuration settings
		return false, nil
	},
	Run: func(state *SetupState) error {
		return nil
	},
}
