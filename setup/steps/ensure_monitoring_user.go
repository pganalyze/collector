package steps

import (
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/setup/query"
	"github.com/pganalyze/collector/setup/state"
)

var EnsureMonitoringUser = &state.Step{
	ID:          "eensure_monitoring_user",
	Description: "Ensure the monitoring user (db_user in the collector config file) exists in Postgres",
	Check: func(s *state.SetupState) (bool, error) {
		pgaUserKey, err := s.CurrentSection.GetKey("db_username")
		if err != nil {
			return false, err
		}
		pgaUser := pgaUserKey.String()

		var result query.Row
		result, err = s.QueryRunner.QueryRow(fmt.Sprintf("SELECT true FROM pg_user WHERE usename = %s", pq.QuoteLiteral(pgaUser)))
		if err == query.ErrNoRows {
			return false, nil
		} else if err != nil {
			return false, err
		}
		return result.GetBool(0), nil
	},
	Run: func(s *state.SetupState) error {
		pgaUserKey, err := s.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := pgaUserKey.String()

		var doCreateUser bool
		if s.Inputs.Scripted {
			if !s.Inputs.EnsureMonitoringUser.Valid ||
				!s.Inputs.EnsureMonitoringUser.Bool {
				return fmt.Errorf("create_monitoring_user flag not set and specified monitoring user %s does not exist", pgaUser)
			}
			doCreateUser = s.Inputs.EnsureMonitoringUser.Bool
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

		return s.QueryRunner.Exec(
			fmt.Sprintf(
				"CREATE USER %s CONNECTION LIMIT 5",
				pq.QuoteIdentifier(pgaUser),
			),
		)
	},
}
