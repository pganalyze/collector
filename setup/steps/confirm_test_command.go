package steps

import (
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	s "github.com/pganalyze/collector/setup/state"
)

var ConfirmTestCommand = &s.Step{
	Description: "Run a test command in Postgres to confirm everything is working",
	Check: func(state *s.SetupState) (bool, error) {
		needsSleep := (state.Inputs.SkipLogInsights.Valid && !state.Inputs.SkipLogInsights.Bool) ||
			(state.Inputs.SkipAutomatedExplain.Valid && !state.Inputs.SkipAutomatedExplain.Bool)
		return !needsSleep || state.DidPgSleep ||
			(state.Inputs.SkipPgSleep.Valid && state.Inputs.SkipPgSleep.Bool), nil
	},
	Run: func(state *s.SetupState) error {
		row, err := state.QueryRunner.QueryRow(
			"SELECT coalesce(max(setting::float), 0) / 1000 * 1.2 from pg_settings where name IN ('log_min_duration_statement', 'auto_explain.log_min_duration')",
		)
		if err != nil {
			return err
		}
		naptime := row.GetFloat(0)
		var doPgSleep bool
		if state.Inputs.Scripted {
			if state.Inputs.SkipPgSleep.Valid {
				doPgSleep = !state.Inputs.SkipPgSleep.Bool
			}
		} else {
			var testCmdType string
			if naptime > 0 {
				testCmdType = "pg_sleep"
			} else {
				testCmdType = "RAISE NOTICE"
			}

			hasAutomatedExplain := state.Inputs.SkipAutomatedExplain.Valid && !state.Inputs.SkipAutomatedExplain.Bool
			var features string
			if hasAutomatedExplain {
				features = "Log Insights and Automated EXPLAIN"
			} else {
				features = "Log Insights"
			}
			err := survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Run a test command (%s) to confirm %s configuration?", testCmdType, features),
				Default: true,
				Help:    "You should see results in pganalyze a few seconds after the query completes",
			}, &doPgSleep)
			if err != nil {
				return err
			}
			state.Inputs.SkipPgSleep = null.BoolFrom(!doPgSleep)
		}

		if !doPgSleep {
			return nil
		}

		var checkStatement string
		if naptime > 0 {
			checkStatement = fmt.Sprintf("SELECT pg_sleep(%f)", naptime)
		} else {
			checkStatement = "DO $$BEGIN RAISE NOTICE 'pganalyze collector test statement'; END$$;"
		}
		err = state.QueryRunner.Exec(checkStatement)
		if err != nil {
			return err
		}
		state.DidPgSleep = true
		return nil
	},
}
