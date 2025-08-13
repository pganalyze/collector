package steps

import (
	"errors"
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/query"
	s "github.com/pganalyze/collector/setup/state"
)

var EnsurePgssExtInstalled = &s.Step{
	ID:          "ensure_pgss_ext_installed",
	Description: "Ensure the pg_stat_statements extension is installed in Postgres",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(
			"SELECT extnamespace::regnamespace::text FROM pg_extension WHERE extname = 'pg_stat_statements'",
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
			if !state.Inputs.EnsurePgStatStatementsInstalled.Valid || !state.Inputs.EnsurePgStatStatementsInstalled.Bool {
				return errors.New("create_pg_stat_statements flag not set and pg_stat_statements does not exist in primary database")
			}
			doCreate = state.Inputs.EnsurePgStatStatementsInstalled.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Create extension pg_stat_statements in public schema for query performance monitoring (will be saved to Postgres)?",
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
