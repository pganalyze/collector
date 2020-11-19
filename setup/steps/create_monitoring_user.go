package steps

import (
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/setup/query"
	s "github.com/pganalyze/collector/setup/state"
)

var CreateMonitoringUser = &s.Step{
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
