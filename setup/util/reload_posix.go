// +build linux freebsd darwin

package util

import (
	"errors"
	"fmt"
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

func ReloadCollector() error {
	processes, err := ps.Processes()
	if err != nil {
		return errors.New("failed to reload collector: could not read process list")
	}
	for _, p := range processes {
		if p.Executable() == "pganalyze-collector" && p.Pid() != os.Getpid() {
			err := reloadPid(p.Pid())
			if err != nil {
				return fmt.Errorf("failed to reload collector: could not send SIGHUP to process: %s", err)
			}
			return nil
		}
	}
	return errors.New("failed to reload: could not find collector in process list; try restarting the pganalyze collector process")
}
