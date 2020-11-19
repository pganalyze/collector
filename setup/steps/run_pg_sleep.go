package steps

import (
	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	s "github.com/pganalyze/collector/setup/state"
)

var RunPgSleep = &s.Step{
	Description: "Run a pg_sleep command to confirm everything is working",
	Check: func(state *s.SetupState) (bool, error) {
		return state.DidPgSleep || (state.Inputs.SkipPgSleep.Valid && state.Inputs.SkipPgSleep.Bool), nil
	},
	Run: func(state *s.SetupState) error {
		var doPgSleep bool
		if state.Inputs.Scripted {
			if state.Inputs.SkipPgSleep.Valid {
				doPgSleep = !state.Inputs.SkipPgSleep.Bool
			}
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Run pg_sleep command to confirm configuration?",
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

		err := state.QueryRunner.Exec(
			"SELECT pg_sleep(max(setting::float) / 1000 * 1.2) from pg_settings where name IN ('log_min_duration_statement', 'auto_explain.log_min_duration')",
		)
		if err != nil {
			return err
		}
		state.DidPgSleep = true
		return nil
	},
}
