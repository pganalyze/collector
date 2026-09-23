//go:build linux || freebsd || darwin
// +build linux freebsd darwin

package util

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/shirou/gopsutil/process"
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

// ReloadSelf - sends SIGHUP to the collector's own process, triggering the
// same reload as a manual one (the running process re-reads the config file
// and restarts collection). Used for automatic reloads when the config file
// has changed and the auto_reload setting is enabled.
func ReloadSelf() error {
	return syscall.Kill(os.Getpid(), syscall.SIGHUP)
}

func Reload() (reloadedPid int, err error) {
	processes, err := process.Processes()
	if err != nil {
		return -1, fmt.Errorf("could not read process list: %s", err)
	}
	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}
		if name == "pganalyze-collector" && int(p.Pid) != os.Getpid() {
			err := reloadPid(int(p.Pid))
			if err != nil {
				return -1, fmt.Errorf("could not send SIGHUP to process: %s", err)
			}
			return int(p.Pid), nil
		}
	}
	return -1, errors.New("could not find collector in process list; try restarting the pganalyze-collector process")
}
