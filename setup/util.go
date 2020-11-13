package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-ini/ini"
	"github.com/lib/pq"
	"github.com/pganalyze/collector/setup/query"
)

func bold(str string) string {
	return fmt.Sprintf("\033[1m%s\033[0m", str)
}

func includes(strings []string, str string) bool {
	for _, s := range strings {
		if s == str {
			return true
		}
	}
	return false
}

type LocalPostgres struct {
	SocketDir string
	LocalAddr string
	Port      int
}

var pgsqlDomainSocketPortRe = regexp.MustCompile("\\d+$")

func getSocketDirMatches(dir string) ([]LocalPostgres, error) {
	var result []LocalPostgres
	// technically this should be a filepath.Join, but Unix-domain sockets do not work
	// on windows anyway
	globPattern := fmt.Sprintf("%s/.s.PGSQL.*", dir)
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		portStr := pgsqlDomainSocketPortRe.FindString(match)
		if portStr == "" {
			continue
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}
		result = append(result, LocalPostgres{SocketDir: dir, Port: port})
	}
	return result, nil
}

func discoverLocalPostgres() ([]LocalPostgres, error) {
	// TODO: find tcp sockets if no unix sockets?
	// TODO: confirm these are live by checking pids?
	varRunMatches, err := getSocketDirMatches("/var/run/postgresql")
	if err != nil {
		return nil, err
	}
	tmpMatches, err := getSocketDirMatches("/tmp")
	if err != nil {
		return nil, err
	}
	var result []LocalPostgres
	return append(append(result, varRunMatches...), tmpMatches...), nil
}

func getPostmasterPid() (int, error) {
	pidStr, err := exec.Command("pgrep", "-U", "postgres", "-o", "postgres").Output()
	if err != nil {
		return -1, fmt.Errorf("failed to find postmaster pid: %s", err)
	}

	pid, err := strconv.Atoi(string(pidStr[:len(pidStr)-1]))
	if err != nil {
		return -1, fmt.Errorf("postmaster pid is not an integer: %s", err)
	}

	return pid, nil
}

func getDataDirectory(postmasterPid int) (string, error) {
	dataDirectory := os.Getenv("PGDATA")
	if dataDirectory != "" {
		return dataDirectory, nil
	}

	dataDirectory, err := filepath.EvalSymlinks("/proc/" + strconv.Itoa(postmasterPid) + "/cwd")
	if err != nil {
		return "", fmt.Errorf("failed to resolve data directory path: %s", err)
	}

	return dataDirectory, nil
}

func discoverLogLocation(config *ini.Section, runner *query.Runner) (string, error) {
	dbHostKey, err := config.GetKey("db_host")
	if err != nil {
		return "", err
	}
	dbHost := dbHostKey.String()
	if dbHost != "localhost" && dbHost != "127.0.0.1" {
		return "", errors.New("detected remote server - Log Insights requires the collector to run on the database server directly for self-hosted systems")
	}

	row, err := runner.QueryRow("SELECT current_setting('log_destination'), current_setting('logging_collector'), current_setting('log_directory')")
	if err != nil {
		return "", err
	}
	logDestination := row.GetString(0)
	loggingCollector := row.GetString(1)
	logDirectory := row.GetString(2)

	if logDestination == "syslog" {
		return "", errors.New("log_destination detected as syslog - please check our setup guide for rsyslogd or syslog-ng instructions")
	} else if logDestination != "stderr" {
		return "", fmt.Errorf("unsupported log_destination %s", logDestination)
	}

	postmasterPid, err := getPostmasterPid()
	if err != nil {
		return "", err
	}
	var logLocation string
	if loggingCollector == "on" {
		if !strings.HasPrefix(logDirectory, "/") {
			dataDir, err := getDataDirectory(postmasterPid)
			if err != nil {
				return "", err
			}

			logDirectory = dataDir + "/" + logDirectory
		}
		logLocation = logDirectory
	} else {
		// assume stdout/stderr redirect to logfile, typical with postgresql-common on Ubuntu/Debian
		logFile, err := filepath.EvalSymlinks("/proc/" + strconv.FormatInt(int64(postmasterPid), 10) + "/fd/1")
		if err != nil {
			return "", err
		}
		// TODO: should we use the directory of this file rather than the file itself?
		logLocation = logFile
	}

	return logLocation, nil
}

func applyConfigSetting(setting, value string, runner *query.Runner) error {
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

func getPendingSharedPreloadLibraries(runner *query.Runner) (string, error) {
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
                    COALESCE(
                      sourcefile, current_setting('data_directory') || '/postgresql.auto.conf'
                    )
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

func getConjuctionList(strs []string) string {
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

func getOptsWithRecommendation(opts []string, recommendedIdx int) []string {
	result := make([]string, len(opts))
	for i, opt := range opts {
		var newOpt string
		if i == recommendedIdx {
			newOpt = fmt.Sprintf("%s (recommended)", opt)
		} else {
			newOpt = opt
		}
		result[i] = newOpt
	}
	return result
}

func usingLogExplain(section *ini.Section) (bool, error) {
	k, err := section.GetKey("enable_log_explain")
	if err != nil {
		return false, err
	}
	return k.Bool()
}

var expectedMd5s = map[string]string{
	"explain":              "814292aad6ba4a207ea7b8c9fb47836b",
	"get_stat_replication": "d4321fedd7286ca0752c6eff13991288",
}

func validateHelperFunction(fn string, runner *query.Runner) (bool, error) {
	// TODO: validating full function definition may be too strict?
	expected, ok := expectedMd5s[fn]
	if !ok {
		return false, fmt.Errorf("unrecognized helper function %s", fn)
	}
	row, err := runner.QueryRow(
		fmt.Sprintf(
			"SELECT md5(prosrc) FROM pg_proc WHERE proname = %s AND pronamespace::regnamespace::text = 'pganalyze'",
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
