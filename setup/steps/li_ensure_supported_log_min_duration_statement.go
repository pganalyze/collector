package steps

import (
	"errors"
	"fmt"
	"strconv"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var ConfigureLogMinDurationStatement = &state.Step{
	ID:          "li_ensure_supported_log_min_duration_statement",
	Kind:        state.LogInsightsStep,
	Description: "Ensure the log_min_duration_statement setting in Postgres is supported by the collector",
	Check: func(s *state.SetupState) (bool, error) {
		row, err := s.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_min_duration_statement'`)
		if err != nil {
			return false, err
		}

		lmdsVal := row.GetInt(0)
		needsUpdate := !isSupportedLmds(lmdsVal) ||
			(s.Inputs.Scripted &&
				s.Inputs.GUCS.LogMinDurationStatement.Valid &&
				int(s.Inputs.GUCS.LogMinDurationStatement.Int64) != lmdsVal)
		return !needsUpdate, nil
	},
	Run: func(s *state.SetupState) error {
		row, err := s.QueryRunner.QueryRow(`SELECT setting FROM pg_settings WHERE name = 'log_min_duration_statement'`)
		if err != nil {
			return err
		}
		oldVal := fmt.Sprintf("%sms", row.GetString(0))

		var newVal string
		if s.Inputs.Scripted {
			if !s.Inputs.GUCS.LogMinDurationStatement.Valid {
				return errors.New("log_min_duration_statement not provided and current value is unsupported")
			}
			newValNum := int(s.Inputs.GUCS.LogMinDurationStatement.Int64)
			if !isSupportedLmds(newValNum) {
				return fmt.Errorf("log_min_duration_statement provided as unsupported value '%d'", newValNum)
			}
			newVal = strconv.Itoa(newValNum)
		} else {
			err = survey.AskOne(&survey.Input{
				Message: fmt.Sprintf(
					"Setting 'log_min_duration_statement' is set to '%s', below supported threshold of 10ms; enter supported value in ms or -1 to disable (will be saved to Postgres):",
					oldVal,
				),
			}, &newVal, survey.WithValidator(util.ValidateLogMinDurationStatement))
			if err != nil {
				return err
			}
		}

		return util.ApplyConfigSetting("log_min_duration_statement", newVal, s.QueryRunner)
	},
}

func isSupportedLmds(value int) bool {
	return value == -1 || value >= logs.MinSupportedLogMinDurationStatement
}
