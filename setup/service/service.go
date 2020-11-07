package service

import (
	"fmt"
	"os/exec"
)

func RestartPostgres() error {
	// TODO: account for different environments; handle error output
	cmd := exec.Command("systemctl", "restart", "postgresql")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart: %s", string(out))
	}
	return nil
}
