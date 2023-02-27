package steps

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/config"
	s "github.com/pganalyze/collector/setup/state"
	mainUtil "github.com/pganalyze/collector/util"
)

var EnsureMonitoringUserPassword = &s.Step{
	ID:          "ensure_monitoring_user_password",
	Description: "Ensure the monitoring user password in Postgres matches db_password in the collector config file",
	Check: func(state *s.SetupState) (bool, error) {
		// We're using config.Read here (and only here) to be able to use the same
		// GetPqOpenString we use in the main collector code
		cfg, err := config.Read(
			&mainUtil.Logger{Destination: log.New(os.Stderr, "", 0)},
			state.ConfigFilename,
		)
		if err != nil {
			return false, err
		}
		if len(cfg.Servers) != 1 {
			return false, fmt.Errorf("expected one server in config; found %d", len(cfg.Servers))
		}
		serverCfg := cfg.Servers[0]
		pqStr, err := serverCfg.GetPqOpenString("", "")
		if err != nil {
			return false, err
		}
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
			if !state.Inputs.EnsureMonitoringPassword.Valid {
				return errors.New("update_monitoring_password flag not set and cannot log in with current credentials")
			}
			doPasswdUpdate = state.Inputs.EnsureMonitoringPassword.Bool
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
				"SET log_statement = none; ALTER USER %s WITH ENCRYPTED PASSWORD %s",
				pq.QuoteIdentifier(pgaUser),
				pq.QuoteLiteral(pgaPasswd),
			),
		)
		return err
	},
}
