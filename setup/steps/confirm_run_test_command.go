package steps

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	s "github.com/pganalyze/collector/setup/state"
)

var ConfirmRunTestCommand = &s.Step{
	ID:          "confirm_run_test_command",
	Description: "Invoke the collector self-test to verify the installation",
	Check: func(state *s.SetupState) (bool, error) {
		return state.DidTestCommand ||
			state.Inputs.Scripted && (!state.Inputs.ConfirmRunTestCommand.Valid || !state.Inputs.ConfirmRunTestCommand.Bool), nil
	},
	Run: func(state *s.SetupState) error {
		var doTestCommand bool
		if state.Inputs.Scripted {
			doTestCommand = state.Inputs.ConfirmRunTestCommand.Valid && state.Inputs.ConfirmRunTestCommand.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "The collector is now ready to begin monitoring. Run test command and reload collector configuration if successful?",
				Default: false,
			}, &doTestCommand)
			if err != nil {
				return err
			}
			state.Inputs.ConfirmRunTestCommand = null.BoolFrom(doTestCommand)
		}
		if !doTestCommand {
			return nil
		}

		state.Log("")
		args := []string{"--test", "--reload", fmt.Sprintf("--config=%s", state.ConfigFilename)}
		extraArgsStr := os.Getenv("PGA_SETUP_COLLECTOR_TEST_EXTRA_ARGS")
		if extraArgsStr != "" {
			extraArgs := strings.Split(extraArgsStr, " ")
			args = append(args, extraArgs...)
		}
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
			return fmt.Errorf("test command failed: %s", addlInfo)
		}
		state.Log("")

		state.DidTestCommand = true
		return nil
	},
}
