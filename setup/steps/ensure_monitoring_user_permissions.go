package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/setup/query"
	s "github.com/pganalyze/collector/setup/state"
)

var EnsureMonitoringUserPermissions = &s.Step{
	Description: "Ensure the monitoring user has sufficient permissions in Postgres for access to queries and monitoring metadata",
	Check: func(state *s.SetupState) (bool, error) {
		pgaUserKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return false, err
		}
		pgaUser := pgaUserKey.String()

		row, err := state.QueryRunner.QueryRow(
			fmt.Sprintf(
				"SELECT usesuper OR pg_has_role(usename, 'pg_monitor', 'usage') FROM pg_user WHERE usename = %s",
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
