package steps

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/go-ini/ini"
	"github.com/pganalyze/collector/setup/query"
	"github.com/pganalyze/collector/setup/state"
	s "github.com/pganalyze/collector/setup/state"
)

var SpecifyDbLogLocation = &s.Step{
	Kind:        state.LogInsightsStep,
	Description: "Specify the location of Postgres log files (db_log_location) in the collector config file",
	Check: func(state *s.SetupState) (bool, error) {
		return state.CurrentSection.HasKey("db_log_location"), nil
	},
	Run: func(state *s.SetupState) error {
		var logLocation string
		if state.Inputs.Scripted {
			loc, err := getLogLocationScripted(state)
			if err != nil {
				return err
			}
			logLocation = loc
		} else {
			loc, err := getLogLocationInteractive(state)
			if err != nil {
				return err
			}
			logLocation = loc
		}

		_, err := state.CurrentSection.NewKey("db_log_location", logLocation)
		if err != nil {
			return err
		}
		return state.SaveConfig()
	},
}

func getLogLocationScripted(state *s.SetupState) (string, error) {
	doGuess := state.Inputs.GuessLogLocation.Valid && state.Inputs.GuessLogLocation.Bool

	if state.Inputs.Settings.DBLogLocation.Valid {
		explicitVal := state.Inputs.Settings.DBLogLocation.String
		if doGuess && explicitVal != "" {
			return "", errors.New("cannot specify both guess_log_location and set explicit db_log_location")
		}
		return explicitVal, nil
	}

	if !doGuess {
		return "", errors.New("db_log_location not provided and guess_log_location flag not set")
	}

	guessedLogLocation, err := discoverLogLocation(state.CurrentSection, state.QueryRunner)
	if err != nil {
		return "", fmt.Errorf("could not determine Postgres log location automatically: %s", err)
	}
	return guessedLogLocation, nil
}

func getLogLocationInteractive(state *s.SetupState) (string, error) {
	guessedLogLocation, err := discoverLogLocation(state.CurrentSection, state.QueryRunner)
	if err != nil {
		state.Verbose("could not determine Postgres log location automatically: %s", err)
	} else {
		var logLocationConfirmed bool
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Your database log file or directory appears to be %s; is this correct (will be saved to collector config)?", guessedLogLocation),
			Default: false,
		}, &logLocationConfirmed)
		if err != nil {
			return "", err
		}
		if logLocationConfirmed {
			return guessedLogLocation, nil
		}
		// otherwise proceed below
	}

	var logLocation string
	err = survey.AskOne(&survey.Input{
		Message: "Please enter the Postgres log file location (will be saved to collector config)",
	}, &logLocation, survey.WithValidator(validatePath))
	return logLocation, err
}

func validatePath(ans interface{}) error {
	ansStr, ok := ans.(string)
	if !ok {
		return errors.New("expected string value")
	}
	// TODO: also confirm this is readable by the regular pganalyze user
	_, err := os.Stat(ansStr)
	if err != nil {
		return err
	}

	return nil
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
	if config.HasKey("db_host") {
		dbHostKey, err := config.GetKey("db_host")
		if err != nil {
			return "", err
		}
		dbHost := dbHostKey.String()
		if dbHost != "localhost" && dbHost != "127.0.0.1" {
			return "", errors.New("detected remote server - Log Insights requires the collector to run on the database server directly for self-hosted systems")
		}
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
