//go:build linux || freebsd || darwin
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

func Reload() (reloadedPid int, err error) {
	processes, err := ps.Processes()
	if err != nil {
		return -1, fmt.Errorf("could not read process list: %s", err)
	}
	for _, p := range processes {
		if p.Executable() == "pganalyze-collector" && p.Pid() != os.Getpid() {
			err := reloadPid(p.Pid())
			if err != nil {
				return -1, fmt.Errorf("could not send SIGHUP to process: %s", err)
			}
			return p.Pid(), nil
		}
	}
	return -1, errors.New("could not find collector in process list; try restarting the pganalyze collector process")
}
