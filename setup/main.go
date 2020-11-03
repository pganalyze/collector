package main

import (
	"database/sql"
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/go-ini/ini"
	"github.com/lib/pq"
	"github.com/shirou/gopsutil/host"
)

type SetupState struct {
	OperatingSystem string
	Platform        string
	PlatformFamily  string
	PlatformVersion string

	SetupConn    *sql.DB
	PGVersionNum int
	PGVersionStr string

	ConfigFilename string
	Config         *ini.File
	CurrentSection *ini.Section

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

func main() {
	steps := []*Step{
		determinePlatform,
		loadConfig,
		establishSuperuserConnection,
		checkPostgresVersion,
		checkReplicationStatus,
		selectDatabases,
		specifyMonitoringUser,
		createMonitoringUser,
		setUpMonitoringUser,
	}

	var setupState SetupState
	// TODO: use same logic as main collector program to set config file location
	setupState.ConfigFilename = "/home/maciek/duboce-labs/collector/pganalyze-collector-autosetup-test.conf"
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

// setup is a series of steps
// each step, we check if done,

// for package installation / init system interaction
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

		return true, nil
	},
}

// for package installation / init system interaction
var loadConfig = &Step{
	Description: "Load pganalyze config",
	Check: func(state *SetupState) (bool, error) {
		// TODO: also stat and check we can write to it?
		config, err := ini.Load(state.ConfigFilename)
		if err != nil {
			return false, err
		}
		// TODO: in theory we could do this for each server instead
		if len(config.Sections()) != 3 {
			// N.B.: DEFAULT section, pganalyze section, server section
			return false, fmt.Errorf("not supported for config file defining more than one server")
		}
		state.Config = config
		for _, section := range config.Sections() {
			if section.Name() == "pganalyze" {
				continue
			} else {
				state.CurrentSection = section
			}
		}

		return true, nil
	},
}

// assume local socket trust auth for postgres user; prompt for credentials if necessary
// N.B.: should make sure this works with cloud provider faux superusers
var establishSuperuserConnection = &Step{
	Description: "Establish Postgres superuser connection",
	Check: func(state *SetupState) (bool, error) {
		if state.SetupConn == nil {
			return false, nil
		}

		err := state.SetupConn.Ping()
		if err != nil {
			return false, err
		}

		return true, nil
	},
	Run: func(state *SetupState) error {
		// TODO: prompt for credentials
		// TODO: right now we assume standard Debian location unix domain socket on 5432; should prompt or attempt more discovery and try /tmp as well
		connStr := "host=/run/postgresql port=5432"
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			return err
		}

		db.SetMaxOpenConns(1)
		state.SetupConn = db

		return nil
	},
}

// this will affect some of the other checks we can run
var checkPostgresVersion = &Step{
	Description: "Check Postgres version",
	Check: func(state *SetupState) (bool, error) {
		err := state.SetupConn.QueryRow("SELECT current_setting('server_version'), current_setting('server_version_num')::integer").Scan(&state.PGVersionStr, &state.PGVersionNum)
		if err != nil {
			return false, err
		}
		return true, nil
	},
}

// is replication target? if so some setup may need to happen on primary instead;
var checkReplicationStatus = &Step{
	Description: "Check replication status",
	Check: func(state *SetupState) (bool, error) {
		var isReplicationTarget bool
		err := state.SetupConn.QueryRow("SELECT pg_is_in_recovery()").Scan(&isReplicationTarget)
		if err != nil {
			return false, err
		}
		if isReplicationTarget {
			return false, errors.New("not supported for replication target")
		}
		return true, nil
	},
}

// get list of databases, set one as primary in config
var selectDatabases = &Step{
	Description: "Select database(s) to monitor",
	Check: func(state *SetupState) (bool, error) {
		return state.CurrentSection.HasKey("db_name"), nil
	},
	Run: func(state *SetupState) error {
		rows, err := state.SetupConn.Query("SELECT datname FROM pg_database WHERE datallowconn AND NOT datistemplate")
		if err != nil {
			return err
		}
		var dbOpts []string
		for rows.Next() {
			var db string
			err = rows.Scan(&db)
			if err != nil {
				return err
			}
			dbOpts = append(dbOpts, db)
		}
		if err = rows.Err(); err != nil {
			return err
		}
		// TODO: deal with db_url

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
		// TODO: deal with db_url
		hasUser := state.CurrentSection.HasKey("db_user")
		return hasUser, nil
	},
	Run: func(state *SetupState) error {
		// TODO: prompt for user
		pgaUser := "pganalyze"
		_, err := state.CurrentSection.NewKey("db_user", pgaUser)
		if err != nil {
			return err
		}
		return state.Config.SaveTo(state.ConfigFilename)
	},
}

var createMonitoringUser = &Step{
	Description: "Ensure monitoring user exists",
	Check: func(state *SetupState) (bool, error) {
		// TODO: deal with db_url
		pgaUserKey, err := state.CurrentSection.GetKey("db_user")
		if err != nil {
			return false, err
		}
		pgaUser := pgaUserKey.String()

		var exists bool
		err = state.SetupConn.QueryRow("SELECT true FROM pg_user WHERE usename = $1", pgaUser).Scan(&exists)
		if err == sql.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return true, nil
	},
	Run: func(state *SetupState) error {
		pgaUserKey, err := state.CurrentSection.GetKey("db_user")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()
		// TODO: generate secure password
		pgaPassword := "hunter2"

		_, err = state.SetupConn.Exec(
			fmt.Sprintf(
				"CREATE USER %s WITH ENCRYPTED PASSWORD %s",
				pq.QuoteIdentifier(pgaUser),
				pq.QuoteLiteral(pgaPassword),
			),
		)
		if err != nil {
			return err
		}

		_, err = state.CurrentSection.NewKey("db_password", pgaPassword)
		if err != nil {
			return err
		}

		return state.Config.SaveTo(state.ConfigFilename)
	},
}

// ensure user has correct permissions
var setUpMonitoringUser = &Step{
	Description: "Set up monitoring user",
	Check: func(state *SetupState) (bool, error) {
		// TODO: deal with db_url
		pgaUserKey, err := state.CurrentSection.GetKey("db_user")
		if err != nil {
			return false, err
		}
		pgaUser := pgaUserKey.String()

		var hasPermissions bool
		// TODO: deal with postgres <10
		// TODO: more lenient check
		err = state.SetupConn.QueryRow("SELECT true FROM pg_user WHERE usename = $1 AND (usesuper OR pg_has_role(usename, 'pg_monitor', 'member'))", pgaUser).Scan(&hasPermissions)
		if err == sql.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return true, nil
	},
	Run: func(state *SetupState) error {
		pgaUserKey, err := state.CurrentSection.GetKey("db_user")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()

		// TODO: deal with postgres <10
		_, err = state.SetupConn.Exec(
			fmt.Sprintf(
				"GRANT pg_monitor to %s",
				pq.QuoteIdentifier(pgaUser),
			),
		)
		return err
	},
}

// check if installed / available; see if postgresql-contrib package is necessary
var setUpPgStatStatements = &Step{}

// check log settings, log file configuration, update config file if necessary
var setUpLogInsights = &Step{
	Description: "log insights setup",
}

// ask if user wants auto_explain or log-based explain (and steer them appropriately)
var setUpExplain = &Step{
	Description: "EXPLAIN setup",
}
