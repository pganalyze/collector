package util

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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

func GetPendingSharedPreloadLibraries(runner *query.Runner) (string, error) {
	// When shared_preload_libraries is updated, since the setting requires a restart for the
	// changes to take effect, the new value is not reflected with SHOW or current_setting().
	// To make sure we don't clobber any pending changes (including our own, if adding both
	// pg_stat_statements *and* auto_explain), we need to read the configured-but-not-yet-applied
	// value from the config file (there does not appear to be a better way to do this)

	// N.B.: we project name here even though we don't explicitly need it, because a valid (and
	// in fact, common) value for shared_preload_libraries is the empty string, and because our
	// query mechanism depends on CSV, and because of https://github.com/golang/go/issues/39119 ,
	// that value cannot be parsed correctly by Go's encoding/csv if that's the only value in the
	// CSV file output.

	// N.B.: note that although pg_settings contains sourcefile and sourceline, these refer
	// to a snapshot of the config file as it existed when the currently-active value of a setting
	// was loaded, which may be totally different from what the config file looks like now. Because
	// of that, we check postgresql.auto.conf first (if it exists and contains the value) and fall
	// back to sourcefile.

	// N.B.: we need IS DISTINCT FROM NULL rather than IS NOT NULL because of the latter's odd behavior
	// with row-valued expressions: https://www.postgresql.org/docs/current/functions-comparison.html#id-1.5.8.8.19.1
	row, err := runner.QueryRow(`
SELECT
  name,
  CASE
    WHEN NOT pending_restart THEN
      setting
    ELSE
      btrim(
        regexp_replace(
          COALESCE(
            (SELECT line FROM
              regexp_split_to_table(
                pg_read_file(
                  CASE
                    WHEN pg_stat_file('postgresql.auto.conf', true) IS DISTINCT FROM NULL THEN
                      'postgresql.auto.conf'
                    ELSE
                      sourcefile
                  END
                ), '\s*$\s*', 'm'
              ) WITH ORDINALITY AS lines(line, line_num)
              WHERE
                line LIKE name || ' = %'
              ORDER BY
                line_num DESC
              LIMIT 1
            ),
            (SELECT line FROM
              regexp_split_to_table(
                pg_read_file(sourcefile), '\s*$\s*', 'm'
              ) WITH ORDINALITY AS lines(line, line_num)
              WHERE
                line LIKE name || ' = %'
              ORDER BY
                line_num DESC
              LIMIT 1
            )
          ),
          name || ' = ', ''
        ),
        ''''
      )
  END AS pending_value
FROM
  pg_settings
WHERE
  name = 'shared_preload_libraries'`)
	if err != nil {
		return "", err
	}
	return row.GetString(1), nil
}

func Includes(strings []string, str string) bool {
	for _, s := range strings {
		if s == str {
			return true
		}
	}
	return false
}

func JoinWithAnd(strs []string) string {
	switch len(strs) {
	case 0:
		return ""
	case 1:
		return strs[0]
	case 2:
		return fmt.Sprintf("%s and %s", strs[0], strs[1])
	default:
		return fmt.Sprintf("%s, and %s", strings.Join(strs[:len(strs)-1], ", "), strs[len(strs)-1])
	}
}
