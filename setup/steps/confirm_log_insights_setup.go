package steps

import (
	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	s "github.com/pganalyze/collector/setup/state"
)

var ConfirmLogInsightsSetup = &s.Step{
	Description: "Check whether Log Insights should be configured",
	Check: func(state *s.SetupState) (bool, error) {
		return state.Inputs.SkipLogInsights.Valid, nil
	},
	Run: func(state *s.SetupState) error {
		var setUpLogInsights bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Proceed to configuring optional Log Insights feature?",
			Help:    "Learn more at https://pganalyze.com/log-insights",
			Default: false,
		}, &setUpLogInsights)
		if err != nil {
			return err
		}
		state.Inputs.SkipLogInsights = null.BoolFrom(!setUpLogInsights)

		return nil
	},
}
