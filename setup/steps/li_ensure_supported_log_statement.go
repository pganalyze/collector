package steps

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsureSupportedLogStatement = &s.Step{
	ID:          "li_ensure_supported_log_statement",
	Kind:        s.LogInsightsStep,
	Description: "Ensure the log_statement setting in Postgres is supported by the collector",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_statement'`)
		if err != nil {
			return false, err
		}
		currValue := row.GetString(0)
		needsUpdate := currValue == "all" ||
			(state.Inputs.Scripted &&
				state.Inputs.GUCS.LogStatement.Valid &&
				currValue != state.Inputs.GUCS.LogStatement.String)

		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		var newVal string
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogStatement.Valid {
				return errors.New("log_statement value not provided and current value not supported")
			}
			if state.Inputs.GUCS.LogStatement.String == "all" {
				return errors.New("log_statement provided as unsupported value 'all'")
			}

			newVal = state.Inputs.GUCS.LogStatement.String
		} else {
			err := survey.AskOne(&survey.Select{
				Message: "Setting 'log_statement' is set to unsupported value 'all'; select supported value (will be saved to Postgres):",
				Options: []string{"none", "ddl", "mod"},
			}, &newVal)
			if err != nil {
				return err
			}
		}

		return util.ApplyConfigSetting("log_statement", newVal, state.QueryRunner)
	},
}
