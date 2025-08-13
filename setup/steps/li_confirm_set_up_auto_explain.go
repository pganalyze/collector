package steps

import (
	"errors"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var ConfirmSetUpAutoExplain = &state.Step{
	ID: "li_confirm_set_up_auto_explain",
	// N.B.: this step, asking the user whether to set up automated explain, is *not* an AutomatedExplainStep
	// itself, but it is a state.LogInsightsStep because it depends on log insights
	Kind:        state.LogInsightsStep,
	Description: "Confirm whether to set up the optional Automated EXPLAIN feature",
	Check: func(s *state.SetupState) (bool, error) {
		// skip the question if we've already answered one way or the other
		if s.Inputs.ConfirmSetUpAutomatedExplain.Valid {
			return true, nil
		}

		// otherwise, definitely prompt if it's not set up
		if !s.CurrentSection.HasKey("enable_log_explain") {
			return false, nil
		}

		isLogExplainKey, err := s.CurrentSection.GetKey("enable_log_explain")
		if err != nil {
			return false, err
		}
		isLogExplain, err := isLogExplainKey.Bool()
		if err != nil {
			return false, err
		}
		if isLogExplain {
			return true, nil
		}

		// assume auto_explain if we got this far
		spl, err := util.GetPendingSharedPreloadLibraries(s.QueryRunner)
		if err != nil {
			return false, err
		}
		return strings.Contains(spl, "auto_explain"), nil
	},
	Run: func(state *state.SetupState) error {
		if state.Inputs.Scripted {
			return errors.New("skip_auto_explain value must be specified")
		}

		state.Log(`
Log Insights and query performance setup is almost complete. You can complete it
now, or proceed to configuring the optional Automated EXPLAIN feature. Automated
EXPLAIN will require either setting up the auto_explain module (recommended) or
creating helper functions in all monitored databases. The auto_explain module has
minimal impact on most query workloads with our recommended settings; we will review
these during setup.

Learn more at https://pganalyze.com/postgres-explain
`)
		var setUpExplain bool
		err := survey.AskOne(&survey.Confirm{
			Message: "Proceed to configuring optional Automated EXPLAIN feature?",
			Default: false,
		}, &setUpExplain)
		if err != nil {
			return err
		}
		state.Inputs.ConfirmSetUpAutomatedExplain = null.BoolFrom(setUpExplain)

		return nil
	},
}
