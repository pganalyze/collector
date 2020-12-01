package steps

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsureSupportedLogDuration = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Ensure the log_duration setting in Postgres is supported by the collector",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_duration'`)
		if err != nil {
			return false, err
		}

		currValue := row.GetString(0)
		needsUpdate := currValue == "on" ||
			(state.Inputs.Scripted && state.Inputs.GUCS.LogDuration.Valid &&
				state.Inputs.GUCS.LogDuration.String != currValue)

		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		var turnOffLogDuration bool
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogDuration.Valid {
				return errors.New("log_duration value not provided and current value not supported")
			}
			if state.Inputs.GUCS.LogDuration.String == "on" {
				return errors.New("log_duration provided as unsupported value 'on'")
			}
			turnOffLogDuration = state.Inputs.GUCS.LogDuration.String == "off"
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Setting 'log_duration' is set to unsupported value 'on'; set to 'off' (will be saved to Postgres)?",
				Default: false,
			}, &turnOffLogDuration)
			if err != nil {
				return err
			}
		}
		if !turnOffLogDuration {
			// technically there is no error to report here; the re-check will fail
			return nil
		}
		return util.ApplyConfigSetting("log_duration", "off", state.QueryRunner)
	},
}
