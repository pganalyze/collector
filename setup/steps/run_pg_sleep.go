package steps

import (
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	s "github.com/pganalyze/collector/setup/state"
)

var RunPgSleep = &s.Step{
	Description: "Run a pg_sleep command to confirm everything is working",
	Check: func(state *s.SetupState) (bool, error) {
		needsSleep := (state.Inputs.SkipLogInsights.Valid && !state.Inputs.SkipLogInsights.Bool) ||
			(state.Inputs.SkipAutomatedExplain.Valid && !state.Inputs.SkipAutomatedExplain.Bool)
		return !needsSleep || state.DidPgSleep ||
			(state.Inputs.SkipPgSleep.Valid && state.Inputs.SkipPgSleep.Bool), nil
	},
	Run: func(state *s.SetupState) error {
		var doPgSleep bool
		if state.Inputs.Scripted {
			if state.Inputs.SkipPgSleep.Valid {
				doPgSleep = !state.Inputs.SkipPgSleep.Bool
			}
		} else {
			hasAutomatedExplain := state.Inputs.SkipAutomatedExplain.Valid && !state.Inputs.SkipAutomatedExplain.Bool
			var features string
			if hasAutomatedExplain {
				features = "Log Insights and Automated EXPLAIN"
			} else {
				features = "Log Insights"
			}
			err := survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Run pg_sleep command to confirm %s configuration?", features),
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
