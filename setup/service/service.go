package service

import (
	"fmt"
	"os/exec"

	s "github.com/pganalyze/collector/setup/state"
)

func RestartPostgres(state *s.SetupState) error {
	return restartPostgresSystemd()
}

func restartPostgresSystemd() error {
	cmd := exec.Command("systemctl", "restart", "postgresql")
	out, err := cmd.CombinedOutput()
	if err != nil {
		var errInfo = err.Error()
		if len(out) > 0 {
			errInfo += "; " + string(out)
		}
		return fmt.Errorf("failed to restart: %s", errInfo)
	}
	return nil
}
