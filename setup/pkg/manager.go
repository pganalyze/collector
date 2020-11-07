package pkg

import (
	"fmt"
	"os/exec"
)

func InstallPgStatStatements() error {
	// TODO: account for different environments
	// TODO: install contrib corresponding to pg version (e.g., if using pgdg packages)
	return installPkg("postgresql-contrib")
}

func InstallAutoExplain() error {
	return installPkg("postgresql-contrib")
}

func installPkg(name string) error {
	cmd := exec.Command("apt", "install", "postgresql-contrib")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install postgresql-contrib: %s", string(out))
	}
	return nil
}
