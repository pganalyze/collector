package steps

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsureSupportedLogDuration = &state.Step{
	ID:          "li_ensure_supported_log_duration",
	Kind:        state.LogInsightsStep,
	Description: "Ensure the log_duration setting in Postgres is supported by the collector",
	Check: func(s *state.SetupState) (bool, error) {
		row, err := s.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_duration'`)
		if err != nil {
			return false, err
		}

		currValue := row.GetString(0)
		needsUpdate := currValue == "on" ||
			(s.Inputs.Scripted && s.Inputs.GUCS.LogDuration.Valid &&
				s.Inputs.GUCS.LogDuration.String != currValue)

		return !needsUpdate, nil
	},
	Run: func(s *state.SetupState) error {
		var turnOffLogDuration bool
		if s.Inputs.Scripted {
			if !s.Inputs.GUCS.LogDuration.Valid {
				return errors.New("log_duration value not provided and current value not supported")
			}
			if s.Inputs.GUCS.LogDuration.String == "on" {
				return errors.New("log_duration provided as unsupported value 'on'")
			}
			turnOffLogDuration = s.Inputs.GUCS.LogDuration.String == "off"
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
		return util.ApplyConfigSetting("log_duration", "off", s.QueryRunner)
	},
}
