package steps

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsureSupportedLogErrorVerbosity = &s.Step{
	ID:          "li_ensure_supported_log_error_verbosity",
	Kind:        state.LogInsightsStep,
	Description: "Ensure the log_error_verbosity setting in Postgres is supported by the collector",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_error_verbosity'`)
		if err != nil {
			return false, err
		}

		currVal := row.GetString(0)
		needsUpdate := currVal == "verbose" ||
			(state.Inputs.Scripted &&
				state.Inputs.GUCS.LogErrorVerbosity.Valid &&
				currVal != state.Inputs.GUCS.LogErrorVerbosity.String)

		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		var newVal string
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogErrorVerbosity.Valid {
				return errors.New("log_error_verbosity value not provided and current value not supported")
			}
			if state.Inputs.GUCS.LogErrorVerbosity.String == "verbose" {
				return errors.New("log_error_verbosity provided as unsupported value 'verbose'")
			}
			newVal = state.Inputs.GUCS.LogErrorVerbosity.String
		} else {
			err := survey.AskOne(&survey.Select{
				Message: "Setting 'log_error_verbosity' is set to unsupported value 'verbose'; select supported value (will be saved to Postgres):",
				Options: []string{"terse", "default"},
			}, &newVal)
			if err != nil {
				return err
			}
		}

		return util.ApplyConfigSetting("log_error_verbosity", newVal, state.QueryRunner)
	},
}
