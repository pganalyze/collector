package steps

import (
	"errors"

	"github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
)

var CheckRestartNeeded = &state.Step{
	ID:          "check_restart_needed",
	Description: "Check whether a Postgres restart will be necessary in a future step to install the collector",
	Check: func(s *state.SetupState) (bool, error) {
		row, err := s.QueryRunner.QueryRow(
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
			s.Log(
				`
NOTICE: A Postgres restart will be required to set up query performance monitoring.
A prompt will ask to confirm the restart before this guided setup performs it.
`,
			)
		} else if !hasAutoExplain {
			s.Log(
				`
NOTICE: A Postgres restart will not be required to set up query performance monitoring.

However, a restart *will* be required for the recommended setup of the Automated EXPLAIN
feature. You can still use the alternative log-based setup to explore the feature without
having to restart Postgres.
`,
			)
		} else {
			s.Log(
				`
NOTICE: A Postgres restart will *not* be required to set up any features.

Your system is ready to configure query performance monitoring, Log Insights, and
Automated EXPLAIN.
`,
			)
		}
		if s.Inputs.Scripted {
			return true, nil
		}

		var doSetup bool
		err = survey.AskOne(&survey.Confirm{
			Message: "Continue with setup?",
			// Default to continue if no restart required
			Default: hasPgss && hasAutoExplain,
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
