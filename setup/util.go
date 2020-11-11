package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	err := runner.Exec(fmt.Sprintf("ALTER SYSTEM SET %s = %s", setting, pq.QuoteLiteral(value)))
	if err != nil {
		return fmt.Errorf("failed to apply setting: %s", err)
	}
	err = runner.Exec("SELECT pg_reload_conf()")
	if err != nil {
		return fmt.Errorf("failed to reload Postgres configuration after applying setting: %s", err)
	}

	return nil
}
