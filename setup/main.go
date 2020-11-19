package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	golog "log"
	"os"
	"strconv"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/go-ini/ini"
	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/shirou/gopsutil/host"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/setup/log"
	"github.com/pganalyze/collector/setup/query"
	"github.com/pganalyze/collector/setup/service"
	"github.com/pganalyze/collector/setup/state"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/steps"
	"github.com/pganalyze/collector/setup/util"
	mainUtil "github.com/pganalyze/collector/util"
)

const defaultConfigFile = "/etc/pganalyze-collector.conf"

func main() {
	steps := []*s.Step{
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
		steps.ConfigureAutoExplain,
		runPgSleep,
	}

	var setupState state.SetupState
	var verbose bool
	var logFile string
	var inputsFile string
	flag.StringVar(&setupState.ConfigFilename, "config", defaultConfigFile, "Specify alternative path for config file")
	flag.BoolVar(&verbose, "verbose", false, "Include verbose logging output")
	flag.StringVar(&logFile, "log", "", "Save output to log file (always includes verbose output)")
	flag.StringVar(&inputsFile, "inputs", "", "JSON file describing answers to all setup prompts")
	flag.Parse()

	logger := log.NewLogger()
	if logFile == "" {
		if verbose {
			logger.VerboseOutput = os.Stdout
		}
	} else {
		log, err := os.Create(logFile)
		if err != nil {
			fmt.Printf("ERROR: could not open log file %s for writes: %s\n", logFile, err)
			os.Exit(1)
		}
		defer log.Close()
		outputBoth := io.MultiWriter(os.Stdout, log)
		logger.StandardOutput = outputBoth
		if verbose {
			logger.VerboseOutput = outputBoth
		} else {
			logger.VerboseOutput = log
		}
	}
	setupState.Logger = &logger

	var inputs state.SetupInputs
	if inputsFile != "" {
		inputsReader, err := os.Open(inputsFile)
		if err != nil {
			setupState.Log("ERROR: could not open inputs file %s: %s", logFile, err)
			os.Exit(1)
		}
		inputsBytes, err := ioutil.ReadAll(inputsReader)
		if err != nil {
			setupState.Log("ERROR: could not open inputs file %s: %s", logFile, err)
			os.Exit(1)
		}
		err = json.Unmarshal(inputsBytes, &inputs)
		if err != nil {
			setupState.Log("ERROR: could not parse inputs file %s: %s", logFile, err)
			os.Exit(1)
		}
		inputs.Scripted = true
		err = os.Stdin.Close()
		if err != nil {
			setupState.Log("ERROR: could not close stdin for scripted input: %s", logFile, err)
			os.Exit(1)
		}
	}
	setupState.Inputs = &inputs

	id := os.Geteuid()
	if id > 0 {
		setupState.Log(`The pganalyze installer must be run with root privileges. It will provide
details on the process and prompt you before making any changes to the
collector config file or your database. If you prefer, you can instead follow
the manual collector install instructions.`)
		os.Exit(1)
	}

	setupState.Log(`Welcome to the pganalyze collector installer!

We will go through a series of steps to set up the collector to monitor your
Postgres database. We will not make any changes to your database or system
without confirmation.

At a high level, we will:

 1. Configure database access and, if necessary, create the pganalyze database user with monitoring-only access
 2. Update the collector configuration file with these settings
 3. Set up the pg_stat_statements extension in your database for query performance monitoring
 4. (Optional) Change log-related configuration settings to enable the pganalyze Log Insights feature
 5. (Optional) Set up EXPLAIN plan collection to enable the pganalyze Automated EXPLAIN feature

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
	if !setupState.Inputs.Scripted {
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
	}

	for _, step := range steps {
		if (step.Kind == state.LogInsightsStep &&
			setupState.Inputs.SkipLogInsights.Valid &&
			setupState.Inputs.SkipLogInsights.Bool) ||
			(step.Kind == state.AutomatedExplainStep &&
				((setupState.Inputs.SkipLogInsights.Valid &&
					setupState.Inputs.SkipLogInsights.Bool) ||
					(setupState.Inputs.SkipAutomatedExplain.Valid &&
						setupState.Inputs.SkipAutomatedExplain.Bool))) {
			continue
		}
		err := doStep(&setupState, step)
		if err != nil {
			if setupState.NeedsReload {
				setupState.Log(`
WARNING: Exiting with pending changes to collector config.

Please run pganalyze-collector --reload to apply these changes.`)
			}
			os.Exit(1)
		}
	}
}

func doStep(setupState *s.SetupState, step *s.Step) error {
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
		setupState.Log("✗ step failed: %s", err)
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

var determinePlatform = &s.Step{
	Description: "Determine platform",
	Check: func(state *s.SetupState) (bool, error) {
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

var loadConfig = &s.Step{
	Description: "Load collector config",
	Check: func(state *s.SetupState) (bool, error) {
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

var saveAPIKey = &s.Step{
	Description: "Add pganalyze API key to collector config",
	Check: func(state *s.SetupState) (bool, error) {
		return state.PGAnalyzeSection.HasKey("api_key"), nil
	},
	Run: func(state *s.SetupState) error {
		apiKey := os.Getenv("PGA_API_KEY")
		var configWriteConfirmed bool

		if state.Inputs.Scripted {
			if state.Inputs.Settings.APIKey.Valid {
				inputsAPIKey := state.Inputs.Settings.APIKey.String
				if apiKey != "" && inputsAPIKey != apiKey {
					state.Log("WARNING: overriding API key from env with API key from inputs file")
				}
				apiKey = inputsAPIKey
				configWriteConfirmed = true
			} else if apiKey == "" {
				return errors.New("no api_key setting specified and PGA_API_KEY not found in env")
			}
		} else if apiKey == "" {
			err := survey.AskOne(&survey.Input{
				Message: "PGA_API_KEY environment variable not found; please enter API key (will be saved to collector config):",
				Help:    "The key can be found on the API keys page for your organization in the pganalyze app",
			}, &apiKey, survey.WithValidator(survey.Required))
			if err != nil {
				return err
			}
			configWriteConfirmed = true
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "PGA_API_KEY found in environment; save to config file?",
				Default: false,
			}, &configWriteConfirmed)
			if err != nil {
				return err
			}
		}
		if !configWriteConfirmed {
			return nil
		}
		_, err := state.PGAnalyzeSection.NewKey("api_key", apiKey)
		if err != nil {
			return err
		}
		return state.SaveConfig()
	},
}

var establishSuperuserConnection = &s.Step{
	Description: "Ensure Postgres superuser connection",
	Check: func(state *s.SetupState) (bool, error) {
		if state.QueryRunner == nil {
			return false, nil
		}
		err := state.QueryRunner.PingSuper()
		return err == nil, err
	},
	Run: func(state *s.SetupState) error {
		localPgs, err := discoverLocalPostgres()
		if err != nil {
			return err
		}
		var selectedPg LocalPostgres
		if state.Inputs.Scripted {
			if !state.Inputs.PGSetupConnPort.Valid {
				return errors.New("no port specified for setup Postgres connection")
			}
			for _, pg := range localPgs {
				if int(state.Inputs.PGSetupConnPort.Int64) == pg.Port &&
					(!state.Inputs.PGSetupConnSocketDir.Valid ||
						state.Inputs.PGSetupConnSocketDir.String == pg.SocketDir) {
					selectedPg = pg
					break
				}
			}
			if selectedPg.Port == 0 {
				var socketDirStr string
				if state.Inputs.PGSetupConnSocketDir.Valid {
					socketDirStr = " in " + state.Inputs.PGSetupConnSocketDir.String
				}

				return fmt.Errorf(
					"no Postgres server found listening on %d%s",
					state.Inputs.PGSetupConnPort.Int64,
					socketDirStr,
				)
			}
		} else {
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
		}

		var pgSuperuser string
		if state.Inputs.Scripted {
			if !state.Inputs.PGSetupConnUser.Valid {
				return errors.New("no user specified for setup Postgres connection")
			}
			pgSuperuser = state.Inputs.PGSetupConnUser.String
		} else {
			err = survey.AskOne(&survey.Select{
				Message: "Select Postgres superuser to connect as for configuration purposes",
				Help:    "We will create a separate, restricted monitoring user for the collector later",
				Options: []string{"postgres", "another user..."},
			}, &pgSuperuser)
			if err != nil {
				return err
			}
			if pgSuperuser != "postgres" {
				err = survey.AskOne(&survey.Input{
					Message: "Enter Postgres superuser to connect as for configuration purposes",
					Help:    "We will create a separate, restricted monitoring user for the collector later",
				}, &pgSuperuser, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}
			}
		}

		state.QueryRunner = query.NewRunner(pgSuperuser, selectedPg.SocketDir, selectedPg.Port)
		return nil
	},
}

var checkPostgresVersion = &s.Step{
	Description: "Check Postgres version",
	Check: func(state *s.SetupState) (bool, error) {
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

var checkReplicationStatus = &s.Step{
	Description: "Check replication status",
	Check: func(state *s.SetupState) (bool, error) {
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

var selectDatabases = &s.Step{
	Description: "Select database(s) to monitor",
	Check: func(state *s.SetupState) (bool, error) {
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
	Run: func(state *s.SetupState) error {
		rows, err := state.QueryRunner.Query("SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate")
		if err != nil {
			return err
		}
		var dbOpts []string
		for _, row := range rows {
			dbOpts = append(dbOpts, row.GetString(0))
		}

		var dbNames []string
		if state.Inputs.Scripted {
			if !state.Inputs.Settings.DBName.Valid {
				return errors.New("no db_name setting specified")
			}
			dbNameInputs := strings.Split(state.Inputs.Settings.DBName.String, ",")
			for i, dbNameInput := range dbNameInputs {
				trimmed := strings.TrimSpace(dbNameInput)
				if trimmed == "*" {
					dbNames = append(dbNames, trimmed)
				} else {
					for _, opt := range dbOpts {
						if trimmed == opt {
							dbNames = append(dbNames, trimmed)
							break
						}
					}
				}

				if len(dbNames) != i+1 {
					return fmt.Errorf("database %s not found", trimmed)
				}
			}
		} else {
			var primaryDb string
			err = survey.AskOne(&survey.Select{
				Message: "Choose a primary database to monitor (will be saved to collector config):",
				Options: dbOpts,
				Help:    "The collector will connect to this database for monitoring; others can be added next",
			}, &primaryDb)
			if err != nil {
				return err
			}

			dbNames = append(dbNames, primaryDb)
			if len(dbOpts) > 0 {
				var otherDbs []string
				for _, db := range dbOpts {
					if db == primaryDb {
						continue
					}
					otherDbs = append(otherDbs, db)
				}
				var othersOpt int
				err = survey.AskOne(&survey.Select{
					Message: "Monitor other databases? (will be saved to collector config):",
					Help:    "The 'all' option will also automatically monitor all future databases created on this server",
					Options: []string{"no other databases", "all other databases", "select databases..."},
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
		}

		dbNamesStr := strings.Join(dbNames, ",")
		_, err = state.CurrentSection.NewKey("db_name", dbNamesStr)
		if err != nil {
			return err
		}

		return state.SaveConfig()
	},
}

var specifyMonitoringUser = &s.Step{
	Description: "Check config for monitoring user",
	Check: func(state *s.SetupState) (bool, error) {
		hasUser := state.CurrentSection.HasKey("db_username")
		return hasUser, nil
	},
	Run: func(state *s.SetupState) error {
		var pgaUser string

		if state.Inputs.Scripted {
			if !state.Inputs.Settings.DBUsername.Valid {
				return errors.New("no db_username setting specified")
			}
			pgaUser = state.Inputs.Settings.DBUsername.String
		} else {
			var monitoringUserIdx int
			err := survey.AskOne(&survey.Select{
				Message: "Select Postgres user for the collector to use (will be saved to collector config):",
				Help:    "If the user does not exist, it can be created in a later step",
				Options: []string{"pganalyze (recommended)", "a different user"},
			}, &monitoringUserIdx)
			if err != nil {
				return err
			}

			if monitoringUserIdx == 0 {
				pgaUser = "pganalyze"
			} else if monitoringUserIdx == 1 {
				err := survey.AskOne(&survey.Input{
					Message: "Enter Postgres user for the collector to use (will be saved to collector config):",
					Help:    "If the user does not exist, it can be created in a later step",
				}, &pgaUser, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}
			} else {
				panic(fmt.Sprintf("unexpected user selection: %d", monitoringUserIdx))
			}
		}

		_, err := state.CurrentSection.NewKey("db_username", pgaUser)
		if err != nil {
			return err
		}
		return state.SaveConfig()
	},
}

var createMonitoringUser = &s.Step{
	Description: "Ensure monitoring user exists",
	Check: func(state *s.SetupState) (bool, error) {
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
	Run: func(state *s.SetupState) error {
		pgaUserKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()

		var doCreateUser bool
		if state.Inputs.Scripted {
			if !state.Inputs.CreateMonitoringUser.Valid ||
				!state.Inputs.CreateMonitoringUser.Bool {
				return fmt.Errorf("create_monitoring_user flag not set and specified monitoring user %s does not exist", pgaUser)
			}
			doCreateUser = state.Inputs.CreateMonitoringUser.Bool
		} else {
			err = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("User %s does not exist in Postgres; create user (will be saved to Postgres)?", pgaUser),
				Help:    "If you skip this step, create the user manually before proceeding",
				Default: false,
			}, &doCreateUser)
			if err != nil {
				return err
			}
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

var configureMonitoringUserPasswd = &s.Step{
	Description: "Configure monitoring user password",
	Check: func(state *s.SetupState) (bool, error) {
		hasPassword := state.CurrentSection.HasKey("db_password")
		return hasPassword, nil
	},
	Run: func(state *s.SetupState) error {
		var passwordStrategy int
		if state.Inputs.Scripted {
			if state.Inputs.GenerateMonitoringPassword.Valid && state.Inputs.GenerateMonitoringPassword.Bool {
				if state.Inputs.Settings.DBPassword.Valid && state.Inputs.Settings.DBPassword.String != "" {
					return errors.New("cannot specify both generate password and set explicit password")
				}
				passwordStrategy = 0
			} else if state.Inputs.Settings.DBPassword.Valid && state.Inputs.Settings.DBPassword.String != "" {
				passwordStrategy = 1
			} else {
				return errors.New("no db_password specified and generate_monitoring_password flag not set")
			}
		} else {
			err := survey.AskOne(&survey.Select{
				Message: "Select how to set up the collector user password (will be saved to collector config):",
				Options: []string{"generate random password (recommended)", "enter existing password"},
			}, &passwordStrategy)
			if err != nil {
				return err
			}
		}

		var pgaPasswd string
		if passwordStrategy == 0 {
			passwdBytes := make([]byte, 16)
			rand.Read(passwdBytes)
			pgaPasswd = hex.EncodeToString(passwdBytes)
		} else if passwordStrategy == 1 {
			if state.Inputs.Scripted {
				pgaPasswd = state.Inputs.Settings.DBPassword.String
			} else {
				err := survey.AskOne(&survey.Input{
					Message: "Enter password for the collector to use (will be saved to collector config):",
				}, &pgaPasswd, survey.WithValidator(survey.Required))
				if err != nil {
					return err
				}
			}
		} else {
			panic(fmt.Sprintf("unexpected password option selection: %d", passwordStrategy))
		}

		_, err := state.CurrentSection.NewKey("db_password", pgaPasswd)
		if err != nil {
			return err
		}

		return state.SaveConfig()
	},
}

var applyMonitoringUserPasswd = &s.Step{
	Description: "Apply monitoring user password",
	Check: func(state *s.SetupState) (bool, error) {
		cfg, err := config.Read(
			&mainUtil.Logger{Destination: golog.New(os.Stderr, "", 0)},
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
	Run: func(state *s.SetupState) error {
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
		if state.Inputs.Scripted {
			if !state.Inputs.UpdateMonitoringPassword.Valid || !state.Inputs.UpdateMonitoringPassword.Bool {
				return errors.New("update_monitoring_password flag not set and cannot log in with current credentials")
			}
			doPasswdUpdate = state.Inputs.UpdateMonitoringPassword.Bool
		} else {
			err = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Update password for user %s with configured value (will be saved to Postgres)?", pgaUser),
				Help:    "If you skip this step, ensure the password matches before proceeding",
			}, &doPasswdUpdate)
			if err != nil {
				return err
			}
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
var setUpMonitoringUser = &s.Step{
	Description: "Set up monitoring user",
	Check: func(state *s.SetupState) (bool, error) {
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
	Run: func(state *s.SetupState) error {
		pgaUserKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()

		// TODO: deal with postgres <10
		var doGrant bool
		if state.Inputs.Scripted {
			if !state.Inputs.SetUpMonitoringUser.Valid || !state.Inputs.SetUpMonitoringUser.Bool {
				return errors.New("set_up_monitoring_user flag not set and monitoring user does not have adequate permissions")
			}
			doGrant = state.Inputs.SetUpMonitoringUser.Bool
		} else {
			err = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Grant role pg_monitor to user %s (will be saved to Postgres)?", pgaUser),
				Help:    "Learn more about pg_monitor here: https://www.postgresql.org/docs/current/default-roles.html",
			}, &doGrant)
			if err != nil {
				return err
			}
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

var createPganalyzeSchema = &s.Step{
	Description: "Create pganalyze schema and helper functions",
	Check: func(state *s.SetupState) (bool, error) {
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
		valid, err := validateHelperFunction("get_stat_replication", state.QueryRunner)
		if err != nil {
			return false, err
		}
		if !valid {
			return false, nil
		}

		return true, nil
	},
	Run: func(state *s.SetupState) error {
		var doSetup bool
		if state.Inputs.Scripted {
			if !state.Inputs.CreateHelperFunctions.Valid || !state.Inputs.CreateHelperFunctions.Bool {
				return errors.New("create_helper_functions flag not set and pganalyze schema or helper functions do not exist")
			}
			doSetup = state.Inputs.CreateHelperFunctions.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Create pganalyze schema and helper functions (will be saved to Postgres)?",
				Default: false,
				// TODO: better link?
				Help: "These helper functions allow the collector to monitor database statistics without being able to read your data; learn more here: https://github.com/pganalyze/collector/#setting-up-a-restricted-monitoring-user",
			}, &doSetup)
			if err != nil {
				return err
			}
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

var checkPgssAvailable = &s.Step{
	Description: "Prepare for pg_stat_statements install",
	Check: func(state *s.SetupState) (bool, error) {
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
	Run: func(state *s.SetupState) error {
		// TODO: install contrib package?
		return errors.New("extension pg_stat_statements is not available")
	},
}

// install pg_stat_statements extension
var createPgss = &s.Step{
	Description: "Install pg_stat_statements",
	Check: func(state *s.SetupState) (bool, error) {
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
	Run: func(state *s.SetupState) error {
		var doCreate bool
		if state.Inputs.Scripted {
			if !state.Inputs.CreatePgStatStatements.Valid || !state.Inputs.CreatePgStatStatements.Bool {
				return errors.New("create_pg_stat_statements flag not set and pg_stat_statements does not exist in primary database")
			}
			doCreate = state.Inputs.CreatePgStatStatements.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Create extension pg_stat_statements in public schema (will be saved to Postgres)?",
				Default: false,
				Help:    "Learn more about pg_stat_statements here: https://www.postgresql.org/docs/current/pgstatstatements.html",
			}, &doCreate)
			if err != nil {
				return err
			}
		}

		if !doCreate {
			return nil
		}
		return state.QueryRunner.Exec("CREATE EXTENSION pg_stat_statements SCHEMA public")
	},
}

var enablePgss = &s.Step{
	Description: "Enable pg_stat_statements",
	Check: func(state *s.SetupState) (bool, error) {
		spl, err := getPendingSharedPreloadLibraries(state.QueryRunner)
		if err != nil {
			return false, err
		}

		return strings.Contains(spl, "pg_stat_statements"), nil
	},
	Run: func(state *s.SetupState) error {
		var doAdd bool
		if state.Inputs.Scripted {
			if !state.Inputs.EnablePgStatStatements.Valid || !state.Inputs.EnablePgStatStatements.Bool {
				return errors.New("enable_pg_stat_statements flag not set but pg_stat_statements not in shared_preload_libraries")
			}
			doAdd = state.Inputs.EnablePgStatStatements.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Add pg_stat_statements to shared_preload_libraries (will be saved to Postgres)?",
				Default: false,
				Help:    "Postgres will have to be restarted in a later step to apply this configuration change; learn more about shared_preload_libraries here: https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SHARED-PRELOAD-LIBRARIES",
			}, &doAdd)
			if err != nil {
				return err
			}
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
		return util.ApplyConfigSetting("shared_preload_libraries", newSpl, state.QueryRunner)
	},
}

var confirmLogInsightsSetup = &s.Step{
	Description: "Check whether Log Insights should be configured",
	Check: func(state *s.SetupState) (bool, error) {
		return state.Inputs.SkipLogInsights.Valid, nil
	},
	Run: func(state *s.SetupState) error {
		var setUpLogInsights bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Proceed to configuring optional Log Insights feature?",
			Help:    "Learn more at https://pganalyze.com/log-insights",
			Default: false,
		}, &setUpLogInsights)
		if err != nil {
			return err
		}
		state.Inputs.SkipLogInsights = null.BoolFrom(!setUpLogInsights)

		return nil
	},
}

var configureLogErrorVerbosity = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Check log_error_verbosity",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_error_verbosity'`)
		if err != nil {
			return false, err
		}

		currVal := row.GetString(0)
		needsUpdate := currVal == "verbose" ||
			(state.Inputs.Scripted &&
				state.Inputs.GUCS.LogErrorVerbosity.Valid &&
				currVal != state.Inputs.GUCS.LogErrorVerbosity.String)

		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		var newVal string
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogErrorVerbosity.Valid {
				return errors.New("log_error_verbosity value not provided and current value not supported")
			}
			newVal = state.Inputs.GUCS.LogErrorVerbosity.String
		} else {
			err := survey.AskOne(&survey.Select{
				Message: "Setting 'log_error_verbosity' is set to unsupported value 'all'; select supported value (will be saved to Postgres):",
				Options: []string{"terse", "default"},
			}, &newVal)
			if err != nil {
				return err
			}
		}

		return util.ApplyConfigSetting("log_error_verbosity", newVal, state.QueryRunner)
	},
}

var configureLogDuration = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Check log_duration",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_duration'`)
		if err != nil {
			return false, err
		}

		currValue := row.GetString(0)
		needsUpdate := currValue == "on" ||
			(state.Inputs.Scripted && state.Inputs.GUCS.LogDuration.Valid &&
				state.Inputs.GUCS.LogDuration.String != currValue)

		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		var turnOffLogDuration bool
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogDuration.Valid {
				return errors.New("log_error_verbosity value not provided and current value not supported")
			}
			turnOffLogDuration = state.Inputs.GUCS.LogDuration.String == "off"
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Setting 'log_duration' is set to unsupported value 'on'; set to 'off' (will be saved to Postgres)?",
				Default: false,
			}, &turnOffLogDuration)
			if err != nil {
				return err
			}
		}
		if !turnOffLogDuration {
			// technically there is no error to report here; the re-check will fail
			return nil
		}
		return util.ApplyConfigSetting("log_duration", "off", state.QueryRunner)
	},
}

var configureLogStatement = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Check log_statement",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_statement'`)
		if err != nil {
			return false, err
		}
		currValue := row.GetString(0)
		needsUpdate := currValue == "all" ||
			(state.Inputs.Scripted &&
				state.Inputs.GUCS.LogStatement.Valid &&
				currValue != state.Inputs.GUCS.LogStatement.String)

		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		var newVal string
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogStatement.Valid {
				return errors.New("log_statement value not provided and current value not supported")
			}
			newVal = state.Inputs.GUCS.LogStatement.String
		} else {
			err := survey.AskOne(&survey.Select{
				Message: "Setting 'log_statement' is set to unsupported value 'all'; select supported value (will be saved to Postgres):",
				Options: []string{"none", "ddl", "mod"},
			}, &newVal)
			if err != nil {
				return err
			}
		}

		return util.ApplyConfigSetting("log_statement", newVal, state.QueryRunner)
	},
}

var configureLogMinDurationStatement = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Check log_min_duration_statement",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_min_duration_statement'`)
		if err != nil {
			return false, err
		}

		lmdsVal := row.GetInt(0)
		needsUpdate := (lmdsVal < 10 && lmdsVal != -1) ||
			(state.Inputs.Scripted &&
				state.Inputs.GUCS.LogMinDurationStatement.Valid &&
				int(state.Inputs.GUCS.LogMinDurationStatement.Int64) != lmdsVal)
		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_min_duration_statement'`)
		if err != nil {
			return err
		}
		oldVal := fmt.Sprintf("%sms", row.GetString(0))

		var newVal string
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogMinDurationStatement.Valid {
				return errors.New("log_min_duration_statement not provided and current value is unsupported")
			}
			newVal = strconv.Itoa(int(state.Inputs.GUCS.LogMinDurationStatement.Int64))
		} else {
			err = survey.AskOne(&survey.Input{
				Message: fmt.Sprintf(
					"Setting 'log_min_duration_statement' is set to '%s', below supported threshold of 10ms; enter supported value in ms or 0 to disable (will be saved to Postgres):",
					oldVal,
				),
			}, &newVal, survey.WithValidator(util.ValidateLogMinDurationStatement))
			if err != nil {
				return err
			}
		}

		return util.ApplyConfigSetting("log_min_duration_statement", newVal, state.QueryRunner)
	},
}

var configureLogLinePrefix = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Check log_line_prefix",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_line_prefix'`)
		if err != nil {
			return false, err
		}

		currValue := row.GetString(0)
		needsUpdate := !includes(s.SupportedLogLinePrefixes, currValue) ||
			(state.Inputs.Scripted &&
				state.Inputs.GUCS.LogLinePrefix.Valid &&
				currValue != state.Inputs.GUCS.LogLinePrefix.String)

		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_line_prefix'`)
		if err != nil {
			return err
		}
		oldVal := row.GetString(0)
		var opts []string
		for i, llp := range s.SupportedLogLinePrefixes {
			// N.B.: we quote the options because many prefixes end in whitespace; we need to make that clear
			var opt string
			if i == 0 {
				opt = fmt.Sprintf("'%s' (recommended)", llp)
			} else {
				opt = fmt.Sprintf("'%s'", llp)
			}
			opts = append(opts, opt)
		}

		var selectedPrefix string
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogLinePrefix.Valid {
				return errors.New("log_line_prefix not provided and current setting is not supported")
			}
			selectedPrefix = state.Inputs.GUCS.LogLinePrefix.String
		} else {
			var prefixIdx int
			err = survey.AskOne(&survey.Select{
				Message: fmt.Sprintf("Setting 'log_line_prefix' is set to unsupported value '%s'; set to (will be saved to Postgres):", oldVal),
				Help:    "Check format specifier reference in Postgres documentation: https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-LINE-PREFIX",
				Options: opts,
			}, &prefixIdx)
			if err != nil {
				return err
			}
			selectedPrefix = s.SupportedLogLinePrefixes[prefixIdx]
		}
		return util.ApplyConfigSetting("log_line_prefix", pq.QuoteLiteral(selectedPrefix), state.QueryRunner)
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

var configureLogLocation = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Determine log location",
	Check: func(state *s.SetupState) (bool, error) {
		return state.CurrentSection.HasKey("db_log_location"), nil
	},
	Run: func(state *s.SetupState) error {
		guessedLogLocation, err := discoverLogLocation(state.CurrentSection, state.QueryRunner)
		if err != nil {
			return err
		}
		var logLocationConfirmed bool
		if state.Inputs.Scripted {
			if state.Inputs.GuessLogLocation.Valid && state.Inputs.GuessLogLocation.Bool {
				if state.Inputs.Settings.DBLogLocation.Valid && state.Inputs.Settings.DBLogLocation.String != "" {
					return errors.New("cannot specify both guess_log_location and set explicit db_log_location")
				}
				logLocationConfirmed = state.Inputs.GuessLogLocation.Bool
			}
		} else {
			err = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Your database log file or directory appears to be %s; is this correct (will be saved to collector config)?", guessedLogLocation),
				Default: false,
			}, &logLocationConfirmed)
			if err != nil {
				return err
			}
		}

		var logLocation string
		if logLocationConfirmed {
			logLocation = guessedLogLocation
		} else if state.Inputs.Scripted {
			if !state.Inputs.Settings.DBLogLocation.Valid || state.Inputs.Settings.DBLogLocation.String == "" {
				return errors.New("db_log_location not provided and guess_log_location flag not set")
			}
			logLocation = state.Inputs.Settings.DBLogLocation.String
		} else {
			err = survey.AskOne(&survey.Input{
				Message: "Please enter the Postgres log file location (will be saved to collector config)",
			}, &logLocation, survey.WithValidator(validatePath))
		}
		_, err = state.CurrentSection.NewKey("db_log_location", logLocation)
		if err != nil {
			return err
		}
		return state.SaveConfig()
	},
}

var confirmAutoExplainSetup = &s.Step{
	// N.B.: this step, asking the user whether to set up automated explain, is *not* an AutomatedExplainStep
	// itself, but it is a state.LogInsightsStep because it depends on log insights
	Kind:        state.LogInsightsStep,
	Description: "Check whether to configure Automated EXPLAIN",
	Check: func(state *s.SetupState) (bool, error) {
		return state.Inputs.SkipAutomatedExplain.Valid, nil
	},
	Run: func(state *s.SetupState) error {
		var setUpExplain bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Proceed to configuring optional Automated EXPLAIN feature?",
			Help:    "Learn more at https://pganalyze.com/postgres-explain",
			Default: false,
		}, &setUpExplain)
		if err != nil {
			return err
		}
		state.Inputs.SkipAutomatedExplain = null.BoolFrom(!setUpExplain)

		return nil
	},
}

var checkUseLogBasedExplain = &s.Step{
	Kind:        s.AutomatedExplainStep,
	Description: "Check whether to use the auto_explain module or Log-based EXPLAIN",
	Check: func(state *s.SetupState) (bool, error) {
		return state.CurrentSection.HasKey("enable_log_explain"), nil
	},
	Run: func(state *s.SetupState) error {
		var useLogBased bool
		if state.Inputs.Scripted {
			if !state.Inputs.UseLogBasedExplain.Valid {
				return errors.New("use_log_based_explain not set")
			}
			useLogBased = state.Inputs.UseLogBasedExplain.Bool
		} else {
			var optIdx int
			err := survey.AskOne(&survey.Select{
				Message: "Select automated EXPLAIN mechanism to use (will be saved to collector config):",
				Help:    "Learn more about the options at https://pganalyze.com/docs/explain/setup",
				Options: []string{"auto_explain (recommended)", "Log-based EXPLAIN"},
			}, &optIdx)
			if err != nil {
				return err
			}
			useLogBased = optIdx == 1
		}

		_, err := state.CurrentSection.NewKey("enable_log_explain", strconv.FormatBool(useLogBased))
		if err != nil {
			return err
		}
		return state.SaveConfig()
	},
}

var createLogExplainHelper = &s.Step{
	Kind:        s.AutomatedExplainStep,
	Description: "Create log-based EXPLAIN helper function",
	Check: func(state *s.SetupState) (bool, error) {
		logExplain, err := util.UsingLogExplain(state.CurrentSection)
		if err != nil {
			return false, err
		}
		if !logExplain {
			return true, nil
		}
		return validateHelperFunction("explain", state.QueryRunner)
	},
	Run: func(state *s.SetupState) error {
		var doCreate bool
		if state.Inputs.Scripted {
			if !state.Inputs.CreateExplainHelper.Valid || !state.Inputs.CreateHelperFunctions.Bool {
				return errors.New("create_explain_helper flag not set and helper function does not exist or does not match expected signature")
			}
			doCreate = state.Inputs.CreateHelperFunctions.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Create (or update) EXPLAIN helper function (will be saved to Postgres)?",
				Default: false,
			}, &doCreate)
			if err != nil {
				return err
			}
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

var checkAutoExplainAvailable = &s.Step{
	Kind:        s.AutomatedExplainStep,
	Description: "Prepare for auto_explain install",
	Check: func(state *s.SetupState) (bool, error) {
		logExplain, err := util.UsingLogExplain(state.CurrentSection)
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
	Run: func(state *s.SetupState) error {
		// TODO: install contrib package?
		return errors.New("module auto_explain is not available")
	},
}

var enableAutoExplain = &s.Step{
	Kind:        s.AutomatedExplainStep,
	Description: "Enable auto_explain",
	Check: func(state *s.SetupState) (bool, error) {
		logExplain, err := util.UsingLogExplain(state.CurrentSection)
		if err != nil || logExplain {
			return logExplain, err
		}
		spl, err := getPendingSharedPreloadLibraries(state.QueryRunner)
		if err != nil {
			return false, err
		}
		return strings.Contains(spl, "auto_explain"), nil
	},
	Run: func(state *s.SetupState) error {
		var doAdd bool
		if state.Inputs.Scripted {
			if !state.Inputs.EnableAutoExplain.Valid || !state.Inputs.EnableAutoExplain.Bool {
				return errors.New("enable_auto_explain flag not set but auto_explain configuration selected")
			}
			doAdd = state.Inputs.EnableAutoExplain.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Add auto_explain to shared_preload_libraries (will be saved to Postgres)?",
				Default: false,
				Help:    "Postgres will have to be restarted in a later step to apply this configuration change; learn more about shared_preload_libraries here: https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-SHARED-PRELOAD-LIBRARIES",
			}, &doAdd)
			if err != nil {
				return err
			}
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
		return util.ApplyConfigSetting("shared_preload_libraries", newSpl, state.QueryRunner)
	},
}

var reloadCollector = &s.Step{
	Description: "Reload collector configuration",
	Check: func(state *s.SetupState) (bool, error) {
		return !state.NeedsReload || state.DidReload, nil
	},
	Run: func(state *s.SetupState) error {
		var doReload bool
		if state.Inputs.Scripted {
			if !state.Inputs.ConfirmCollectorReload.Valid || !state.Inputs.ConfirmCollectorReload.Bool {
				return errors.New("confirm_collector_reload flag not set but collector reload required")
			}
			doReload = state.Inputs.ConfirmCollectorReload.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "The collector configuration must be reloaded for changes to take effect; reload now?",
				Default: false,
			}, &doReload)
			if err != nil {
				return err
			}
		}
		if !doReload {
			return nil
		}
		err := reload()
		if err != nil {
			return err
		}
		state.DidReload = true
		return nil
	},
}

var restartPg = &s.Step{
	Description: "If necessary, restart Postgres to have configuration changes take effect",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow("SELECT COUNT(*) FROM pg_settings WHERE pending_restart;")
		if err != nil {
			return false, err
		}
		return row.GetInt(0) == 0, nil
	},
	Run: func(state *s.SetupState) error {
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
		if state.Inputs.Scripted {
			if !state.Inputs.ConfirmPostgresRestart.Valid || !state.Inputs.ConfirmPostgresRestart.Bool {
				return errors.New("confirm_postgres_restart flag not set but Postgres restart required")
			}
			restartNow = state.Inputs.ConfirmPostgresRestart.Bool
		} else {
			err = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("WARNING: Postgres must be restarted for changes to %s to take effect; restart Postgres now?", pendingList),
				Default: false,
			}, &restartNow)
			if err != nil {
				return err
			}

			if !restartNow {
				return nil
			}

			err = survey.AskOne(&survey.Confirm{
				Message: "WARNING: Your database will be restarted. Are you sure?",
				Default: false,
			}, &restartNow)
			if err != nil {
				return err
			}
		}

		if !restartNow {
			return nil
		}

		return service.RestartPostgres()
	},
}

var runPgSleep = &s.Step{
	Description: "Run a pg_sleep command to confirm everything is working",
	Check: func(state *s.SetupState) (bool, error) {
		return state.DidPgSleep || (state.Inputs.SkipPgSleep.Valid && state.Inputs.SkipPgSleep.Bool), nil
	},
	Run: func(state *s.SetupState) error {
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
			state.Inputs.SkipPgSleep = null.BoolFrom(true)
			return nil
		}

		err = state.QueryRunner.Exec(
			"SELECT pg_sleep(max(setting::float) / 1000 * 1.2) from pg_settings where name IN ('log_min_duration_statement', 'auto_explain.log_min_duration')",
		)
		if err != nil {
			return err
		}
		state.DidPgSleep = true
		return nil
	},
}
