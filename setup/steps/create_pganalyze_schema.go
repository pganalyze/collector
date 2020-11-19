package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/lib/pq"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var CreatePganalyzeSchema = &s.Step{
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
		valid, err := util.ValidateHelperFunction("get_stat_replication", state.QueryRunner)
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
