package steps

import (
	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/setup/state"
	s "github.com/pganalyze/collector/setup/state"
)

var ConfirmAutoExplainSetup = &s.Step{
	// N.B.: this step, asking the user whether to set up automated explain, is *not* an AutomatedExplainStep
	// itself, but it is a state.LogInsightsStep because it depends on log insights
	Kind:        state.LogInsightsStep,
	Description: "Check whether to configure Automated EXPLAIN",
	Check: func(state *s.SetupState) (bool, error) {
		return state.Inputs.SkipAutomatedExplain.Valid, nil
	},
	Run: func(state *s.SetupState) error {
		var setUpExplain bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Proceed to configuring optional Automated EXPLAIN feature?",
			Help:    "Learn more at https://pganalyze.com/postgres-explain",
			Default: false,
		}, &setUpExplain)
		if err != nil {
			return err
		}
		state.Inputs.SkipAutomatedExplain = null.BoolFrom(!setUpExplain)

		return nil
	},
}
