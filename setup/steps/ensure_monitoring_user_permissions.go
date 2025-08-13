package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/setup/query"
	"github.com/pganalyze/collector/setup/state"
)

var EnsureMonitoringUserPermissions = &state.Step{
	ID:          "ensure_monitoring_user_permissions",
	Description: "Ensure the monitoring user has sufficient permissions in Postgres for access to queries and monitoring metadata",
	Check: func(s *state.SetupState) (bool, error) {
		pgaUserKey, err := s.CurrentSection.GetKey("db_username")
		if err != nil {
			return false, err
		}
		pgaUser := pgaUserKey.String()

		row, err := s.QueryRunner.QueryRow(
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
	Run: func(s *state.SetupState) error {
		pgaUserKey, err := s.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()

		var doGrant bool
		if s.Inputs.Scripted {
			if !s.Inputs.EnsureMonitoringPermissions.Valid || !s.Inputs.EnsureMonitoringPermissions.Bool {
				return errors.New("set_up_monitoring_user flag not set and monitoring user does not have adequate permissions")
			}
			doGrant = s.Inputs.EnsureMonitoringPermissions.Bool
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

		return s.QueryRunner.Exec(
			fmt.Sprintf(
				"GRANT pg_monitor to %s",
				pq.QuoteIdentifier(pgaUser),
			),
		)
	},
}
