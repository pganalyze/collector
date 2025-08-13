package steps

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/setup/state"
)

var ConfirmEmitTestExplain = &state.Step{
	Kind:        state.AutomatedExplainStep,
	ID:          "ae_confirm_emit_test_explain",
	Description: "Invoke the collector EXPLAIN test to generate an EXPLAIN plan based on pg_sleep",
	Check: func(s *state.SetupState) (bool, error) {
		return s.DidTestExplainCommand ||
			s.Inputs.Scripted && (!s.Inputs.ConfirmRunTestExplainCommand.Valid || !s.Inputs.ConfirmRunTestExplainCommand.Bool), nil
	},
	Run: func(s *state.SetupState) error {
		var doTestCommand bool
		if s.Inputs.Scripted {
			doTestCommand = s.Inputs.ConfirmRunTestExplainCommand.Valid && s.Inputs.ConfirmRunTestExplainCommand.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Issue pg_sleep statement on server to test EXPLAIN configuration",
				Help:    "Learn more about pg_sleep here: https://www.postgresql.org/docs/current/functions-datetime.html#FUNCTIONS-DATETIME-DELAY",
				Default: false,
			}, &doTestCommand)
			if err != nil {
				return err
			}
			s.Inputs.ConfirmRunTestExplainCommand = null.BoolFrom(doTestCommand)
		}
		if !doTestCommand {
			return nil
		}

		s.Log("")
		args := []string{"--test-explain", fmt.Sprintf("--config=%s", s.ConfigFilename)}
		cmd := exec.Command("pganalyze-collector", args...)
		var stdOut bytes.Buffer
		cmd.Stdout = &stdOut
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			addlInfo := err.Error()
			stdOutStr := stdOut.String()
			if stdOutStr != "" {
				addlInfo = addlInfo + "\n" + stdOutStr
			}
			return fmt.Errorf("test explain command failed: %s", addlInfo)
		}
		s.Log("")

		s.DidTestExplainCommand = true
		return nil
	},
}
