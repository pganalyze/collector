// +build linux freebsd darwin

package util

import (
	"os"
	"os/exec"
	"strconv"

	"github.com/keybase/go-ps"
)

func reloadPid(pid int) error {
	var err error
	kill, err := exec.LookPath("kill")
	if err != nil {
		return err
	}
	cmd := exec.Command(kill, "-s", "HUP", strconv.Itoa(pid))
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func Reload(logger *Logger) {
	processes, err := ps.Processes()
	if err != nil {
		logger.PrintError("Error: Failed to reload collector: Could not read process list\n")
		os.Exit(1)
	}
	for _, p := range processes {
		if p.Executable() == "pganalyze-collector" && p.Pid() != os.Getpid() {
			err := reloadPid(p.Pid())
			if err != nil {
				logger.PrintError("Error: Failed to reload collector: Could not send SIGHUP to process: %s\n", err)
				os.Exit(1)
			}
			logger.PrintInfo("Successfully reloaded pganalyze collector (PID %d)\n", p.Pid())
			os.Exit(0)
		}
	}
	logger.PrintError("Error: Failed to reload collector: Could not find pganalyze collector in process list\n")
	os.Exit(1)
}
