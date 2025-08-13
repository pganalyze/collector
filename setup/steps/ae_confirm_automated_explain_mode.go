package steps

import (
	"errors"
	"strconv"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
)

var ConfirmAutomatedExplainMode = &state.Step{
	Kind:        state.AutomatedExplainStep,
	ID:          "ae_confirm_automated_explain_mode",
	Description: "Confirm whether to implement Automated EXPLAIN via the recommended auto_explain module or the alternative log-based EXPLAIN",
	Check: func(s *state.SetupState) (bool, error) {
		return s.CurrentSection.HasKey("enable_log_explain"), nil
	},
	Run: func(s *state.SetupState) error {
		var useLogBased bool
		if s.Inputs.Scripted {
			if !s.Inputs.UseLogBasedExplain.Valid {
				return errors.New("use_log_based_explain not set")
			}
			useLogBased = s.Inputs.UseLogBasedExplain.Bool
		} else {
			var optIdx int
			err := survey.AskOne(&survey.Select{
				Message: "Select automated EXPLAIN mechanism to use (will be saved to collector config):",
				Help:    "Learn more about the options at https://pganalyze.com/docs/explain/setup",
				Options: []string{"auto_explain (recommended)", "Log-based EXPLAIN"},
			}, &optIdx)
			if err != nil {
				return err
			}
			useLogBased = optIdx == 1
		}

		_, err := s.CurrentSection.NewKey("enable_log_explain", strconv.FormatBool(useLogBased))
		if err != nil {
			return err
		}
		return s.SaveConfig()
	},
}
