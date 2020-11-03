package steps

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	s "github.com/pganalyze/collector/setup/state"
)

var ConfirmEmitTestExplain = &s.Step{
	Kind:        s.AutomatedExplainStep,
	ID:          "ae_confirm_emit_test_explain",
	Description: "Invoke the collector EXPLAIN test to generate an EXPLAIN plan based on pg_sleep",
	Check: func(state *s.SetupState) (bool, error) {
		return state.DidTestExplainCommand ||
			state.Inputs.Scripted && (!state.Inputs.ConfirmRunTestExplainCommand.Valid || !state.Inputs.ConfirmRunTestExplainCommand.Bool), nil
	},
	Run: func(state *s.SetupState) error {
		var doTestCommand bool
		if state.Inputs.Scripted {
			doTestCommand = state.Inputs.ConfirmRunTestExplainCommand.Valid && state.Inputs.ConfirmRunTestExplainCommand.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "Issue pg_sleep statement on server to test EXPLAIN configuration",
				Help:    "Learn more about pg_sleep here: https://www.postgresql.org/docs/current/functions-datetime.html#FUNCTIONS-DATETIME-DELAY",
				Default: false,
			}, &doTestCommand)
			if err != nil {
				return err
			}
			state.Inputs.ConfirmRunTestExplainCommand = null.BoolFrom(doTestCommand)
		}
		if !doTestCommand {
			return nil
		}

		state.Log("")
		args := []string{"--test-explain", fmt.Sprintf("--config=%s", state.ConfigFilename)}
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
		state.Log("")

		state.DidTestExplainCommand = true
		return nil
	},
}
