package steps

import (
	"errors"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var EnsureAutoExplainInSpl = &state.Step{
	Kind:        state.AutomatedExplainStep,
	ID:          "aemod_ensure_auto_explain_in_spl",
	Description: "Ensure the auto_explain module is included in the shared_preload_libraries setting in Postgres",
	Check: func(s *state.SetupState) (bool, error) {
		logExplain, err := util.UsingLogExplain(s.CurrentSection)
		if err != nil || logExplain {
			return logExplain, err
		}
		spl, err := util.GetPendingSharedPreloadLibraries(s.QueryRunner)
		if err != nil {
			return false, err
		}
		return strings.Contains(spl, "auto_explain"), nil
	},
	Run: func(s *state.SetupState) error {
		var doAdd bool
		if s.Inputs.Scripted {
			if !s.Inputs.EnsureAutoExplainLoaded.Valid || !s.Inputs.EnsureAutoExplainLoaded.Bool {
				return errors.New("enable_auto_explain flag not set but auto_explain configuration selected")
			}
			doAdd = s.Inputs.EnsureAutoExplainLoaded.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Add auto_explain to shared_preload_libraries (will be saved to Postgres--requires restart in a later step)?",
				Default: false,
				Help:    "Postgres will have to be restarted in a later step to apply this configuration change; learn more about Automated EXPLAIN at https://pganalyze.com/postgres-explain",
			}, &doAdd)
			if err != nil {
				return err
			}
		}
		if !doAdd {
			return nil
		}

		existingSpl, err := util.GetPendingSharedPreloadLibraries(s.QueryRunner)
		if err != nil {
			return err
		}
		var newSpl string
		if existingSpl == "" {
			newSpl = "auto_explain"
		} else {
			newSpl = existingSpl + ",auto_explain"
		}
		return util.ApplyConfigSetting("shared_preload_libraries", newSpl, s.QueryRunner)
	},
}
