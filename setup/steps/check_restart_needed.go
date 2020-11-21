package steps

import (
	"errors"

	"github.com/AlecAivazis/survey/v2"
	s "github.com/pganalyze/collector/setup/state"
)

var CheckRestartNeeded = &s.Step{
	Description: "Check whether a Postgres restart will be necessary to install",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(
			`SELECT
current_setting('shared_preload_libraries') LIKE '%pg_stat_statements%',
current_setting('shared_preload_libraries') LIKE '%auto_explain%'`,
		)
		if err != nil {
			return false, err
		}
		hasPgss := row.GetBool(0)
		hasAutoExplain := row.GetBool(1)
		if !hasPgss {
			state.Log(
				`
NOTICE: A Postgres restart will be required to set up query performance monitoring.
A prompt will ask to confirm the restart before this installer performs it.
`,
			)
		} else if !hasAutoExplain {
			state.Log(
				`
NOTICE: A Postgres restart will not be required to set up query performance monitoring.

A restart *will* be required for the recommended setup of the Automated EXPLAIN feature,
but you can use the alternative log-based setup to explore the feature without having to
restart Postgres.
`,
			)
		}
		if state.Inputs.Scripted {
			return true, nil
		}

		var doSetup bool
		err = survey.AskOne(&survey.Confirm{
			Message: "Continue with setup?",
			Default: false,
		}, &doSetup)
		if err != nil {
			return false, err
		}
		if !doSetup {
			return false, errors.New("setup aborted")
		}
		return true, nil
	},
}
