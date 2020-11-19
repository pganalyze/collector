package util

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/go-ini/ini"
	"github.com/pganalyze/collector/setup/query"
)

func UsingLogExplain(section *ini.Section) (bool, error) {
	k, err := section.GetKey("enable_log_explain")
	if err != nil {
		return false, err
	}
	return k.Bool()
}

func ApplyConfigSetting(setting, value string, runner *query.Runner) error {
	// N.B.: we don't quote the value because in the case of lists (like shared_preload_libraries)
	// that does not parse the list correctly
	err := runner.Exec(fmt.Sprintf("ALTER SYSTEM SET %s = %s", setting, value))
	if err != nil {
		return fmt.Errorf("failed to apply setting: %s", err)
	}
	err = runner.Exec("SELECT pg_reload_conf()")
	if err != nil {
		return fmt.Errorf("failed to reload Postgres configuration after applying setting: %s", err)
	}

	return nil
}

func ValidateLogMinDurationStatement(ans interface{}) error {
	ansStr, ok := ans.(string)
	if !ok {
		return errors.New("expected string value")
	}
	ansNum, err := strconv.Atoi(ansStr)
	if err != nil {
		return errors.New("value must be numeric")
	}
	if ansNum < 10 && ansNum != -1 {
		return errors.New("value must be either -1 to disable or 10 or greater")
	}
	return nil
}
