package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
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

	SkipRevokePublicSchema             bool
	SkipAutoExplainRecommendedSettings bool
	SkipPgSleep                        bool

	DidReload  bool
	DidPgSleep bool

	Logger *Logger
}

func (state *SetupState) Log(line string, params ...interface{}) error {
	return state.Logger.Log(line, params...)
}

func (state *SetupState) Verbose(line string, params ...interface{}) error {
	return state.Logger.Verbose(line, params...)
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
		createPganalyzeSchema,
		// TODO: this does not work right now--we check with has_schema_privilege, and after
		// a revoke, the user still has the privilege
		//revokePrivilegesFromMonitoringUser,
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
		checkUseLogBasedExplain,
		createLogExplainHelper,
		checkAutoExplainAvailable,
		enableAutoExplain,

		reloadCollector,
		restartPg,
		configureAutoExplain,
		runPgSleep,
	}

	var setupState SetupState
	var verbose bool
	var logFile string
	flag.StringVar(&setupState.ConfigFilename, "config", defaultConfigFile, "Specify alternative path for config file")
	flag.BoolVar(&verbose, "verbose", false, "Include verbose logging output")
	flag.StringVar(&logFile, "log", "", "Save output to log file (always includes verbose output)")
	flag.Parse()
	var logger Logger
	if logFile != "" {
		logger = NewLogger()
		log, err := os.Create(logFile)
		if err != nil {
			fmt.Errorf("could not open %s for writes: %s", logFile, err)
			return
		}
		defer log.Close()
		outputBoth := io.MultiWriter(os.Stdout, log)
		logger.StandardOutput = outputBoth
		if verbose {
			logger.VerboseOutput = outputBoth
		} else {
			logger.VerboseOutput = log
		}
	} else {
		logger = NewLogger()
		if verbose {
			logger.VerboseOutput = os.Stdout
		}
	}
	setupState.Logger = &logger

	// TODO: check for root?

	setupState.Log(`Welcome to the pganalyze collector installer!

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

Changes to Postgres configuration settings will be done with the ALTER SYSTEM command.
If you later need to refine any of these, make sure to use ALTER SYSTEM or ALTER SYSTEM RESET,
since otherwise, the ALTER SYSTEM changes will override any direct config file edits. Learn
more at https://www.postgresql.org/docs/current/sql-altersystem.html .

You can stop at any time by pressing Ctrl+C.

If you stop before completing setup, you can resume by running the installer
again. We can pick up where you left off.`)
	setupState.Log("")
	var doSetup bool
	err := survey.AskOne(&survey.Confirm{
		Message: "Continue with setup?",
		Default: false,
	}, &doSetup)
	if err != nil {
		setupState.Log("  automated setup failed: %s", err)
	}
	if !doSetup {
		setupState.Log("Exiting...")
		os.Exit(0)
	}

	for _, step := range steps {
		err := doStep(&setupState, step)
		if err != nil {
			os.Exit(1)
		}
	}
}

func doStep(setupState *SetupState, step *Step) error {
	if step.Check == nil {
		panic("step missing completion check")
	}
	setupState.Logger.StartStep(step.Description)
	defer setupState.Logger.EndStep()
	done, err := step.Check(setupState)
	if err != nil {
		setupState.Log("✗ failed to check status: %s", err)
		return err
	}
	if done {
		setupState.Verbose("✓ no changes needed")
		return nil
	}
	if step.Run == nil {
		// panic because we should always define a Run func if a check can fail
		panic("check failed and no resolution defined")
	}
	setupState.Verbose("? suggesting resolution")

	err = step.Run(setupState)
	if err != nil {
		return err
	}

	setupState.Verbose("  re-checking...")
	done, err = step.Check(setupState)
	if err != nil {
		setupState.Log("✗ failed to check status: %s", err)
		return err
	}
	if !done {
		err := errors.New("check still failed after running resolution; please try again")
		setupState.Log("✗ %s", err)
		return err
	}
	setupState.Verbose("✓ step completed")
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
			return false, errors.New("not supported on platforms other than Ubuntu 20.04")
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
		localPgs, err := discoverLocalPostgres()
		if err != nil {
			return err
		}
		var selectedPg LocalPostgres
		if len(localPgs) == 0 {
			return errors.New("failed to find a running local Postgres install")
		} else if len(localPgs) == 1 {
			selectedPg = localPgs[0]
		} else {
			var opts []string
			for _, localPg := range localPgs {
				opts = append(opts, fmt.Sprintf("port %d in socket dir %s", localPg.Port, localPg.SocketDir))
			}
			var selectedIdx int
			err := survey.AskOne(&survey.Select{
				Message: "Found several Postgres installations; please select one",
				Options: opts,
			}, &selectedIdx)
			if err != nil {
				return err
			}
			selectedPg = localPgs[selectedIdx]
		}
		var pgSuperuser string
		err = survey.AskOne(&survey.Select{
			Message: "Select Postgres superuser to connect as for configuration purposes",
			Help:    "We will create a separate, restricted monitoring user for the collector later",
			Options: []string{"postgres", "another user"},
		}, &pgSuperuser)
		if err != nil {
			return err
		}
		if pgSuperuser != "postgres" {
			err = survey.AskOne(&survey.Input{
				Message: "Enter Postgres superuser to connect as for configuration purposes",
				Help:    "We will create a separate monitoring user for the collector later",
			}, &pgSuperuser, survey.WithValidator(survey.Required))
			if err != nil {
				return err
			}
		}

		state.QueryRunner = query.NewRunner(pgSuperuser, selectedPg.SocketDir, selectedPg.Port)
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

		if state.PGVersionNum < 100000 {
			return false, fmt.Errorf("not supported for Postgres versions older than 10; found %s", state.PGVersionStr)
		}

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
		hasDb := state.CurrentSection.HasKey("db_name")
		if !hasDb {
			return false, nil
		}
		key, err := state.CurrentSection.GetKey("db_name")
		if err != nil {
			return false, err
		}
		dbs := key.Strings(",")
		if len(dbs) == 0 || dbs[0] == "" {
			return false, nil
		}
		db := dbs[0]
		// Now that we know the database, connect to the right one for setup:
		// this is important for extensions and helper functions. Note that we
		// need to do this in Check, rather than the Run, since a subsequent
		// execution, resuming an incomplete setup, will not run Run again
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

		var primaryDb string
		err = survey.AskOne(&survey.Select{
			Message: "Choose a primary database to monitor (will be saved to collector config):",
			Options: dbOpts,
			Help:    "The collector will connect to this database for monitoring; others can be added next",
		}, &primaryDb)
		if err != nil {
			return err
		}
		var otherDbs []string
		for _, db := range dbOpts {
			if db == primaryDb {
				continue
			}
			otherDbs = append(otherDbs, db)
		}
		var dbNames []string = []string{primaryDb}
		if len(otherDbs) > 0 {
			var othersOpt int
			err = survey.AskOne(&survey.Select{
				Message: "Monitor other databases? (will be saved to collector config):",
				Options: []string{"no other databases", "all other databases", "select databases"},
			}, &othersOpt)
			if err != nil {
				return err
			}
			if othersOpt == 1 {
				dbNames = append(dbNames, "*")
			} else if othersOpt == 2 {
				var otherDbsSelected []string
				err = survey.AskOne(&survey.MultiSelect{
					Message: "Select other databases to monitor (will be saved to collector config):",
					Options: otherDbs,
				}, &otherDbsSelected)
				if err != nil {
					return err
				}
				dbNames = append(dbNames, otherDbsSelected...)
			}
		}

		_, err = state.CurrentSection.NewKey("db_name", strings.Join(dbNames, ","))
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
				"CREATE USER %s CONNECTION LIMIT 5",
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
		if err != nil {
			isAuthErr := strings.Contains(err.Error(), "authentication failed")
			if isAuthErr {
				return false, nil
			}
			return false, err
		}

		return true, nil

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
			Message: fmt.Sprintf("Update password for user %s with configured value (will be saved to Postgres)?", pgaUser),
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

var createPganalyzeSchema = &Step{
	Description: "Create pganalyze schema and helper functions",
	Check: func(state *SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow("SELECT COUNT(*) FROM pg_namespace WHERE nspname = 'pganalyze'")
		if err != nil {
			return false, err
		}
		count := row.GetInt(0)
		if count != 1 {
			return false, nil
		}
		userKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return false, err
		}
		pgaUser := userKey.String()
		row, err = state.QueryRunner.QueryRow(fmt.Sprintf("SELECT has_schema_privilege(%s, 'pganalyze', 'USAGE')", pq.QuoteLiteral(pgaUser)))
		if err != nil {
			return false, err
		}
		hasUsage := row.GetBool(0)
		if !hasUsage {
			return false, nil
		}
		row, err = state.QueryRunner.QueryRow("SELECT COUNT(*) FROM pg_proc WHERE proname = 'get_stat_replication' AND pronargs = 0")
		if err != nil {
			return false, err
		}
		count = row.GetInt(0)
		if count != 1 {
			return false, nil
		}
		return true, nil
	},
	Run: func(state *SetupState) error {
		var doSetup bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Create pganalyze schema and helper functions (will be saved to Postgres)?",
			Default: false,
			// TODO: better link?
			Help: "These helper functions allow the collector to monitor database statistics without being able to read your data; learn more here: https://github.com/pganalyze/collector/#setting-up-a-restricted-monitoring-user",
		}, &doSetup)
		if err != nil {
			return err
		}
		if !doSetup {
			return nil
		}
		return state.QueryRunner.Exec(`CREATE SCHEMA IF NOT EXISTS pganalyze;
GRANT USAGE ON SCHEMA pganalyze TO pganalyze;

CREATE OR REPLACE FUNCTION pganalyze.get_stat_replication() RETURNS SETOF pg_stat_replication AS
$$
	/* pganalyze-collector */ SELECT * FROM pg_catalog.pg_stat_replication;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;`)
	},
}

var revokePrivilegesFromMonitoringUser = &Step{
	Description: "Ensure the monitoring user has no unnecessary database privileges",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipRevokePublicSchema {
			return true, nil
		}
		pgaKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return false, err
		}
		pgaUser := pgaKey.String()

		row, err := state.QueryRunner.QueryRow(
			fmt.Sprintf(
				"SELECT has_schema_privilege(%s, 'public', 'CREATE') OR has_schema_privilege(%[1]s, 'public', 'USAGE')",
				pq.QuoteLiteral(pgaUser),
			),
		)
		if err != nil {
			return false, err
		}
		return !row.GetBool(0), nil
	},
	Run: func(state *SetupState) error {
		pgaKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := pgaKey.String()
		var doRevoke bool
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Revoke privileges on public schema from %s user (will be saved to Postgres)?", pgaUser),
			Default: true,
			Help:    "The collector does not need this access; we recommend revoking these privileges",
		}, &doRevoke)
		if err != nil {
			return err
		}
		if !doRevoke {
			state.SkipRevokePublicSchema = true
			return nil
		}
		return state.QueryRunner.Exec(fmt.Sprintf("REVOKE ALL ON SCHEMA public FROM %s", pq.QuoteIdentifier(pgaUser)))
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
		spl, err := getPendingSharedPreloadLibraries(state.QueryRunner)
		if err != nil {
			return false, err
		}

		return strings.Contains(spl, "pg_stat_statements"), nil
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

		existingSpl, err := getPendingSharedPreloadLibraries(state.QueryRunner)
		if err != nil {
			return err
		}

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
	if ansNum < 10 && ansNum != -1 {
		return errors.New("value must be either -1 to disable or 10 or greater")
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
		}, &prefixIdx)
		if err != nil {
			return err
		}
		selectedPrefix := supportedLogLinePrefixes[prefixIdx]
		return applyConfigSetting("log_line_prefix", pq.QuoteLiteral(selectedPrefix), state.QueryRunner)
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

var checkUseLogBasedExplain = &Step{
	Description: "Check whether to use the auto_explain module or log-based EXPLAIN",
	Check: func(state *SetupState) (bool, error) {
		return state.CurrentSection.HasKey("enable_log_explain"), nil
	},
	Run: func(state *SetupState) error {
		var optIdx int
		err := survey.AskOne(&survey.Select{
			Message: "Select automated EXPLAIN mechanism to use (will be saved to collector config):",
			Help:    "Learn more about the options at https://pganalyze.com/docs/explain/setup",
			Options: []string{"auto_explain (recommended)", "log-based EXPLAIN"},
		}, &optIdx)
		if err != nil {
			return err
		}
		useLogBased := optIdx == 1
		_, err = state.CurrentSection.NewKey("enable_log_explain", strconv.FormatBool(useLogBased))
		if err != nil {
			return err
		}
		return state.Config.SaveTo(state.ConfigFilename)
	},
}

var createLogExplainHelper = &Step{
	Description: "Create log-based EXPLAIN helper function",
	Check: func(state *SetupState) (bool, error) {
		logExplain, err := usingLogExplain(state.CurrentSection)
		if err != nil {
			return false, err
		}
		if !logExplain {
			return true, nil
		}
		return validateHelperFunction("explain", state.QueryRunner)
	},
	Run: func(state *SetupState) error {
		var doCreate bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Create (or update) EXPLAIN helper function (will be saved to Postgres)?",
			Default: false,
		}, &doCreate)
		if err != nil {
			return err
		}
		if !doCreate {
			return nil
		}
		return state.QueryRunner.Exec(`CREATE OR REPLACE FUNCTION pganalyze.explain(query text, params text[]) RETURNS text AS
$$
DECLARE
	prepared_query text;
	prepared_params text;
	result text;
BEGIN
	SELECT regexp_replace(query, ';+\s*\Z', '') INTO prepared_query;
	IF prepared_query LIKE '%;%' THEN
		RAISE EXCEPTION 'cannot run EXPLAIN when query contains semicolon';
	END IF;

	IF array_length(params, 1) > 0 THEN
		SELECT string_agg(quote_literal(param) || '::unknown', ',') FROM unnest(params) p(param) INTO prepared_params;

		EXECUTE 'PREPARE pganalyze_explain AS ' || prepared_query;
		BEGIN
			EXECUTE 'EXPLAIN (VERBOSE, FORMAT JSON) EXECUTE pganalyze_explain(' || prepared_params || ')' INTO STRICT result;
		EXCEPTION WHEN OTHERS THEN
			DEALLOCATE pganalyze_explain;
			RAISE;
		END;
		DEALLOCATE pganalyze_explain;
	ELSE
		EXECUTE 'EXPLAIN (VERBOSE, FORMAT JSON) ' || prepared_query INTO STRICT result;
	END IF;

	RETURN result;
END
$$ LANGUAGE plpgsql VOLATILE SECURITY DEFINER;`)
	},
}

var checkAutoExplainAvailable = &Step{
	Description: "Prepare for auto_explain install",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipAutomatedExplain {
			return true, nil
		}
		logExplain, err := usingLogExplain(state.CurrentSection)
		if err != nil || logExplain {
			return logExplain, err
		}
		err = state.QueryRunner.Exec("LOAD 'auto_explain'")
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
		logExplain, err := usingLogExplain(state.CurrentSection)
		if err != nil || logExplain {
			return logExplain, err
		}
		spl, err := getPendingSharedPreloadLibraries(state.QueryRunner)
		if err != nil {
			return false, err
		}
		return strings.Contains(spl, "auto_explain"), nil
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

		existingSpl, err := getPendingSharedPreloadLibraries(state.QueryRunner)
		if err != nil {
			return err
		}
		var newSpl string
		if existingSpl == "" {
			newSpl = "auto_explain"
		} else {
			newSpl = existingSpl + ",auto_explain"
		}
		return applyConfigSetting("shared_preload_libraries", newSpl, state.QueryRunner)
	},
}

var reloadCollector = &Step{
	Description: "Reload collector configuration",
	Check: func(state *SetupState) (bool, error) {
		// N.B.: there's no way to tell whether the collector actually needs to reload; we
		// force this once per setup helper invocation (this starts out as false and is set
		// to true below)
		return state.DidReload, nil
	},
	Run: func(state *SetupState) error {
		var doReload bool
		err := survey.AskOne(&survey.Confirm{
			Message: "The collector configuration must be reloaded for changes to take effect; reload now?",
			Default: false,
		}, &doReload)
		if err != nil {
			return err
		}
		if !doReload {
			return nil
		}
		err = reload()
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

// N.B.: this needs to happen *after* the Postgres restart so that ALTER SYSTEM
// recognizes these as valid configuration settings
var configureAutoExplain = &Step{
	Description: "Review auto_explain settings",
	Check: func(state *SetupState) (bool, error) {
		if state.SkipAutomatedExplain {
			return true, nil
		}
		logExplain, err := usingLogExplain(state.CurrentSection)
		if err != nil || logExplain {
			return logExplain, err
		}
		if state.SkipAutoExplainRecommendedSettings {
			return true, nil
		}

		return false, nil
	},
	Run: func(state *SetupState) error {
		var doReview bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Review auto_explain configuration settings?",
			Default: false,
			Help:    "Optional, but will ensure best balance of monitoring visibility and performance; review these settings at https://www.postgresql.org/docs/current/auto-explain.html#id-1.11.7.13.5",
		}, &doReview)
		if err != nil {
			return err
		}
		if !doReview {
			state.SkipAutoExplainRecommendedSettings = true
			return nil
		}

		var logAnalyzeIdx int
		var logAnalyzeOpts = []string{"on", "off"}
		err = survey.AskOne(&survey.Select{
			Message: "Set auto_explain.log_analyze to (will be saved to Postgres):",
			Help:    "Include EXPLAIN ANALYZE output rather than just EXPLAIN output when a plan is logged; required on for several other settings",
			Options: getOptsWithRecommendation(logAnalyzeOpts, 0),
		}, &logAnalyzeIdx)
		if err != nil {
			return err
		}
		logAnalyze := logAnalyzeOpts[logAnalyzeIdx]
		err = applyConfigSetting("auto_explain.log_analyze", logAnalyze, state.QueryRunner)
		if err != nil {
			return err
		}

		if logAnalyze == "on" {
			var logBuffersIdx int
			var logBuffersOpts = []string{"on", "off"}
			err = survey.AskOne(&survey.Select{
				Message: "Set auto_explain.log_buffers to (will be saved to Postgres):",
				Help:    "Include BUFFERS usage information when a plan is logged",
				Options: getOptsWithRecommendation(logBuffersOpts, 0),
			}, &logBuffersIdx)
			if err != nil {
				return err
			}
			logBuffers := logBuffersOpts[logBuffersIdx]
			err = applyConfigSetting("auto_explain.log_buffers", logBuffers, state.QueryRunner)
			if err != nil {
				return err
			}

			var logTimingIdx int
			var logTimingOpts = []string{"on", "off"}
			err = survey.AskOne(&survey.Select{
				Message: "Set auto_explain.log_timing to (will be saved to Postgres):",
				Help:    "Include timing information for each plan node when a plan is logged; can have high performance impact",
				Options: getOptsWithRecommendation(logTimingOpts, 1),
			}, &logTimingIdx)
			if err != nil {
				return err
			}
			logTiming := logTimingOpts[logTimingIdx]
			err = applyConfigSetting("auto_explain.log_timing", logTiming, state.QueryRunner)
			if err != nil {
				return err
			}

			var logTriggersIdx int
			var logTriggersOpts = []string{"on", "off"}
			err = survey.AskOne(&survey.Select{
				Message: "Set auto_explain.log_triggers to (will be saved to Postgres):",
				Help:    "Include trigger execution statistics when a plan is logged",
				Options: getOptsWithRecommendation(logTriggersOpts, 0),
			}, &logTriggersIdx)
			if err != nil {
				return err
			}
			logTriggers := logTriggersOpts[logTriggersIdx]
			err = applyConfigSetting("auto_explain.log_triggers", logTriggers, state.QueryRunner)
			if err != nil {
				return err
			}

			var logVerboseIdx int
			var logVerboseOpts = []string{"on", "off"}
			err = survey.AskOne(&survey.Select{
				Message: "Set auto_explain.log_verbose to (will be saved to Postgres):",
				Help:    "Include VERBOSE EXPLAIN details when a plan is logged",
				Options: getOptsWithRecommendation(logVerboseOpts, 0),
			}, &logVerboseIdx)
			if err != nil {
				return err
			}
			logVerbose := logVerboseOpts[logVerboseIdx]
			err = applyConfigSetting("auto_explain.log_verbose", logVerbose, state.QueryRunner)
			if err != nil {
				return err
			}
		}

		var logFormatIdx int
		var logFormatOpts = []string{"text", "json"}
		err = survey.AskOne(&survey.Select{
			Message: "Set auto_explain.log_format to (will be saved to Postgres):",
			Help:    "Select EXPLAIN output format to be used (only text and json are supported; text format is currently experimental)",
			Options: getOptsWithRecommendation(logFormatOpts, 1),
		}, &logFormatIdx)
		if err != nil {
			return err
		}
		logFormat := logFormatOpts[logFormatIdx]
		err = applyConfigSetting("auto_explain.log_format", logFormat, state.QueryRunner)
		if err != nil {
			return err
		}

		var logMinDuration int
		err = survey.AskOne(&survey.Input{
			Message: "Set auto_explain.log_min_duration, in milliseconds, to (will be saved to Postgres):",
			Help:    "Threshold to log EXPLAIN plans; recommend 1000, must be at least 10",
		}, &logMinDuration, survey.WithValidator(validateLmds))
		if err != nil {
			return err
		}
		err = applyConfigSetting("auto_explain.log_min_duration", strconv.Itoa(logMinDuration), state.QueryRunner)
		if err != nil {
			return err
		}

		var logNestedIdx int
		var logNestedOpts = []string{"on", "off"}
		err = survey.AskOne(&survey.Select{
			Message: "Set auto_explain.log_nested_statements to (will be saved to Postgres):",
			Help:    "Consider statements executed inside functions for logging",
			Options: getOptsWithRecommendation(logNestedOpts, 0),
		}, &logNestedIdx)
		if err != nil {
			return err
		}
		logNested := logNestedOpts[logNestedIdx]
		err = applyConfigSetting("auto_explain.log_nested_statements", logNested, state.QueryRunner)
		return err
	},
}

var runPgSleep = &Step{
	Description: "Run a pg_sleep command to confirm everything is working",
	Check: func(state *SetupState) (bool, error) {
		return state.SkipPgSleep || state.DidPgSleep, nil
	},
	Run: func(state *SetupState) error {
		var doPgSleep bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Run pg_sleep command to confirm configuration?",
			Default: true,
			Help:    "You should see results in pganalyze a few seconds after the query completes",
		}, &doPgSleep)
		if err != nil {
			return err
		}
		if !doPgSleep {
			state.SkipPgSleep = true
			return nil
		}
		row, err := state.QueryRunner.QueryRow(
			"SELECT max(setting::float) / 1000 * 1.2 from pg_settings where name IN ('log_min_duration_statement', 'auto_explain.log_min_duration')",
		)
		if err != nil {
			return err
		}
		sleepDuration := row.GetFloat(0)
		err = state.QueryRunner.Exec(fmt.Sprintf("SELECT pg_sleep(%f)", sleepDuration))
		if err != nil {
			return err
		}
		state.DidPgSleep = true
		return nil
	},
}
