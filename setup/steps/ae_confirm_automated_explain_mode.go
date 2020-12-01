package steps

import (
	"errors"
	"strconv"

	survey "github.com/AlecAivazis/survey/v2"
	s "github.com/pganalyze/collector/setup/state"
)

var ConfirmAutomatedExplainMode = &s.Step{
	Kind:        s.AutomatedExplainStep,
	Description: "Confirm whether to implement Automated EXPLAIN via the recommended auto_explain module or the alternative log-based EXPLAIN",
	Check: func(state *s.SetupState) (bool, error) {
		return state.CurrentSection.HasKey("enable_log_explain"), nil
	},
	Run: func(state *s.SetupState) error {
		var useLogBased bool
		if state.Inputs.Scripted {
			if !state.Inputs.UseLogBasedExplain.Valid {
				return errors.New("use_log_based_explain not set")
			}
			useLogBased = state.Inputs.UseLogBasedExplain.Bool
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

		_, err := state.CurrentSection.NewKey("enable_log_explain", strconv.FormatBool(useLogBased))
		if err != nil {
			return err
		}
		return state.SaveConfig()
	},
}
