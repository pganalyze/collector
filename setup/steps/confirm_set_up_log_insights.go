package steps

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/setup/state"
)

var ConfirmSetUpLogInsights = &state.Step{
	ID:          "confirm_set_up_log_insights",
	Description: "Confirm whether to set up the optional Log Insights feature",
	Check: func(s *state.SetupState) (bool, error) {
		return s.Inputs.ConfirmSetUpLogInsights.Valid || s.PGAnalyzeSection.HasKey("db_log_location"), nil
	},
	Run: func(s *state.SetupState) error {
		if s.Inputs.Scripted {
			return errors.New("skip_log_insights value must be specified")
		}
		s.Log(`
Basic setup is almost complete. You can complete it now, or proceed to
configuring the optional Log Insights feature. Log Insights will require
specifying your database log file (we may be able to detect this), and
may require changes to some logging-related settings.

Setting up Log Insights is required for the Automated EXPLAIN feature.

Learn more at https://pganalyze.com/log-insights
`)
		var setUpLogInsights bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Proceed to configuring optional Log Insights feature?",
			Default: false,
		}, &setUpLogInsights)
		if err != nil {
			return err
		}
		s.Inputs.ConfirmSetUpLogInsights = null.BoolFrom(setUpLogInsights)

		return nil
	},
}
