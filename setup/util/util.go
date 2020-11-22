package util

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-ini/ini"
	"github.com/lib/pq"
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

var expectedMd5s = map[string]string{
	"explain":              "7a0a1784d170975d8538d3b8b38c3fad",
	"get_stat_replication": "066680efec598232c0245477976a2c3d",
}

func ValidateHelperFunction(fn string, runner *query.Runner) (bool, error) {
	// TODO: validating full function definition may be too strict?
	expected, ok := expectedMd5s[fn]
	if !ok {
		return false, fmt.Errorf("unrecognized helper function %s", fn)
	}
	row, err := runner.QueryRow(
		fmt.Sprintf(
			"SELECT md5(btrim(prosrc, E' \\n\\r\\t')) FROM pg_proc WHERE proname = %s AND pronamespace::regnamespace::text = 'pganalyze'",
			pq.QuoteLiteral(fn),
		),
	)
	if err == query.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	actual := row.GetString(0)
	return actual == expected, nil
}

func GetPendingSharedPreloadLibraries(runner *query.Runner) (string, error) {
	// When shared_preload_libraries is updated, since the setting requires a restart for the
	// changes to take effect, the new value is not reflected with SHOW or current_setting().
	// To make sure we don't clobber any pending changes (including our own, if adding both
	// pg_stat_statements *and* auto_explain), we need to read the configured-but-not-yet-applied
	// value from the config file (there does not appear to be a better way to do this)

	// N.B.: we project name here even though we don't explicitly need it,
	// because a valid (and in fact, common) value for shared_preload_libraries
	// is the empty string, and because our query mechanism depends on CSV, and
	// because of https://github.com/golang/go/issues/39119 , that value cannot
	// be parsed correctly by Go's encoding/csv if that's the only value in the
	// CSV file output.

	// N.B.: note also that checking sourcefile/sourceline here is a heuristic: these
	// describe where the *current* value comes from (not the pending value), but this
	// is our best guess.
	row, err := runner.QueryRow(`
SELECT
  name,
  CASE
    WHEN pending_restart THEN
      left(
        right(
          regexp_replace(
            (SELECT line FROM
              (SELECT row_number() OVER () AS line_no, line FROM
                regexp_split_to_table(
                  pg_read_file(
                    COALESCE(sourcefile, 'postgresql.auto.conf')
                  ), '\s*$\s*', 'm'
                ) AS lines(line)
              ) AS numbered_lines(line_no, line)
             WHERE
               CASE WHEN sourceline IS NULL THEN line LIKE name || ' = %' ELSE line_no = sourceline END
            ),
            name || ' = ', ''
          ),
      -1),
    -1)
    ELSE current_setting(name)
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
