package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsurePganalyzeSchema = &s.Step{
	Description: "Ensure the pganalyze schema exists and db_user in the collector config file has USAGE privilege on it",
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
		valid, err := util.ValidateHelperFunction(util.GetStatReplicationHelper, state.QueryRunner)
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
			if !state.Inputs.EnsureHelperFunctions.Valid || !state.Inputs.EnsureHelperFunctions.Bool {
				return errors.New("create_helper_functions flag not set and pganalyze schema or helper functions do not exist")
			}
			doSetup = state.Inputs.EnsureHelperFunctions.Bool
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

		userKey, err := state.CurrentSection.GetKey("db_username")
		if err != nil {
			return err
		}
		pgaUser := userKey.String()

		return state.QueryRunner.Exec(
			fmt.Sprintf(
				`CREATE SCHEMA IF NOT EXISTS pganalyze; GRANT USAGE ON SCHEMA pganalyze TO %s;`,
				pq.QuoteIdentifier(pgaUser),
			) + util.GetStatReplicationHelper.GetDefinition(),
		)
	},
}
