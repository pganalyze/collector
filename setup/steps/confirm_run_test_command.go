package steps

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/guregu/null"
	"github.com/pganalyze/collector/setup/state"
)

var ConfirmRunTestCommand = &state.Step{
	ID:          "confirm_run_test_command",
	Description: "Invoke the collector self-test to verify the installation",
	Check: func(s *state.SetupState) (bool, error) {
		return s.DidTestCommand ||
			s.Inputs.Scripted && (!s.Inputs.ConfirmRunTestCommand.Valid || !s.Inputs.ConfirmRunTestCommand.Bool), nil
	},
	Run: func(s *state.SetupState) error {
		var doTestCommand bool
		if s.Inputs.Scripted {
			doTestCommand = s.Inputs.ConfirmRunTestCommand.Valid && s.Inputs.ConfirmRunTestCommand.Bool
		} else {
			err := survey.AskOne(&survey.Confirm{
				Message: "The collector is now ready to begin monitoring. Run test command and reload collector configuration if successful?",
				Default: false,
			}, &doTestCommand)
			if err != nil {
				return err
			}
			s.Inputs.ConfirmRunTestCommand = null.BoolFrom(doTestCommand)
		}
		if !doTestCommand {
			return nil
		}

		s.Log("")
		args := []string{"--test", "--reload", fmt.Sprintf("--config=%s", s.ConfigFilename)}
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
		s.Log("")

		s.DidTestCommand = true
		return nil
	},
}
