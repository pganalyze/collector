package steps

import (
	"errors"
	"fmt"
	"strconv"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var ConfigureLogMinDurationStatement = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Check log_min_duration_statement",
	Check: func(state *s.SetupState) (bool, error) {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_min_duration_statement'`)
		if err != nil {
			return false, err
		}

		lmdsVal := row.GetInt(0)
		needsUpdate := (lmdsVal < 10 && lmdsVal != -1) ||
			(state.Inputs.Scripted &&
				state.Inputs.GUCS.LogMinDurationStatement.Valid &&
				int(state.Inputs.GUCS.LogMinDurationStatement.Int64) != lmdsVal)
		return !needsUpdate, nil
	},
	Run: func(state *s.SetupState) error {
		row, err := state.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_min_duration_statement'`)
		if err != nil {
			return err
		}
		oldVal := fmt.Sprintf("%sms", row.GetString(0))

		var newVal string
		if state.Inputs.Scripted {
			if !state.Inputs.GUCS.LogMinDurationStatement.Valid {
				return errors.New("log_min_duration_statement not provided and current value is unsupported")
			}
			newVal = strconv.Itoa(int(state.Inputs.GUCS.LogMinDurationStatement.Int64))
		} else {
			err = survey.AskOne(&survey.Input{
				Message: fmt.Sprintf(
					"Setting 'log_min_duration_statement' is set to '%s', below supported threshold of 10ms; enter supported value in ms or 0 to disable (will be saved to Postgres):",
					oldVal,
				),
			}, &newVal, survey.WithValidator(util.ValidateLogMinDurationStatement))
			if err != nil {
				return err
			}
		}

		return util.ApplyConfigSetting("log_min_duration_statement", newVal, state.QueryRunner)
	},
}
