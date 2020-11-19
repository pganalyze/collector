package steps

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var ReloadCollector = &s.Step{
	Description: "Reload collector configuration",
	Check: func(state *s.SetupState) (bool, error) {
		return !state.NeedsReload || state.DidReload, nil
	},
	Run: func(state *s.SetupState) error {
		var doReload bool
		if state.Inputs.Scripted {
			if !state.Inputs.ConfirmCollectorReload.Valid || !state.Inputs.ConfirmCollectorReload.Bool {
				return errors.New("confirm_collector_reload flag not set but collector reload required")
			}
			doReload = state.Inputs.ConfirmCollectorReload.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "The collector configuration must be reloaded for changes to take effect; reload now?",
				Default: false,
			}, &doReload)
			if err != nil {
				return err
			}
		}
		if !doReload {
			return nil
		}
		err := util.ReloadCollector()
		if err != nil {
			return err
		}
		state.DidReload = true
		return nil
	},
}
