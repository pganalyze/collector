package steps

import (
	"errors"
	"strings"

	s "github.com/pganalyze/collector/setup/state"
	"github.com/pganalyze/collector/setup/util"
)

var ConfirmAutoExplainAvailable = &s.Step{
	Kind:        s.AutomatedExplainStep,
	ID:          "aemod_check_auto_explain_available",
	Description: "Confirm the auto_explain contrib module is available",
	Check: func(state *s.SetupState) (bool, error) {
		logExplain, err := util.UsingLogExplain(state.CurrentSection)
		if err != nil || logExplain {
			return logExplain, err
		}
		err = state.QueryRunner.Exec("LOAD 'auto_explain'")
		if err != nil {
			if strings.Contains(err.Error(), "No such file or directory") {
				return false, nil
			}

			return false, err
		}
		return true, err
	},
	Run: func(state *s.SetupState) error {
		return errors.New("contrib module auto_explain is not available")
	},
}
