package selfhosted

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

const settingValueSQL string = `
SELECT setting
	FROM pg_settings
 WHERE name = '%s'`

func getPostgresSetting(settingName string, server state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger) (string, error) {
	var value string

	db, err := postgres.EstablishConnection(server, prefixedLogger, globalCollectionOpts, "")
	if err != nil {
		return "", fmt.Errorf("Could not connect to database to retrieve \"%s\": %s", settingName, err)
	}

	err = db.QueryRow(postgres.QueryMarkerSQL + fmt.Sprintf(settingValueSQL, settingName)).Scan(&value)
	db.Close()
	if err != nil {
		return "", fmt.Errorf("Could not read \"%s\" setting: %s", settingName, err)
	}

	return value, nil
}

// DiscoverLogLocation - Tries to find the log location for a currently running Postgres
// process and outputs the presumed location using the logger
func DiscoverLogLocation(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		if server.Config.DbHost != "localhost" && server.Config.DbHost != "127.0.0.1" {
			prefixedLogger.PrintError("ERROR - Detected remote server - Log Insights requires the collector to run on the database server directly for self-hosted systems")
			continue
		}

		logDestination, err := getPostgresSetting("log_destination", server, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("ERROR - %s", err)
			continue
		}

		if logDestination == "syslog" {
			prefixedLogger.PrintInfo("Log location detected as syslog - please check our setup guide for rsyslogd or syslog-ng instructions")
			continue
		} else if logDestination != "stderr" {
			prefixedLogger.PrintError("ERROR - Unsupported log_destination \"%s\"", logDestination)
			continue
		}

		loggingCollector, err := getPostgresSetting("logging_collector", server, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("ERROR - %s", err)
			continue
		}

		var status helperStatus
		statusBytes, err := exec.Command("/usr/bin/pganalyze-collector-helper", "status").Output()
		if err != nil {
			prefixedLogger.PrintError("ERROR - Could not run helper process: %s", err)
			continue
		} else {
			err = json.Unmarshal(statusBytes, &status)
			if err != nil {
				prefixedLogger.PrintVerbose("ERROR - Could not unmarshal helper status: %s", err)
				continue
			}
		}

		if loggingCollector == "on" {
			logDirectory, err := getPostgresSetting("log_directory", server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
				continue
			}
			prefixedLogger.PrintInfo("Found Log Location: %s", status.DataDirectory+"/"+logDirectory)
			// TODO: log_file_mode is relevant (should be "0640" instead of "0600", and then add the pganalyze user to the postgres group)
		} else { // typical with postgresql-common on Ubuntu/Debian, Docker setup with ports bound to host
			prefixedLogger.PrintInfo("Discovering log directory using open files in postmaster (PID %d)...", status.PostmasterPid)
			logFile, err := filepath.EvalSymlinks("/proc/" + strconv.FormatInt(int64(status.PostmasterPid), 10) + "/fd/1")
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
				continue
			}
			prefixedLogger.PrintInfo("Found Log Location: %s", logFile)
			// TODO: Verify permissions
		}
	}
}

// SetupLogTails - Sets up continuously running log tails for all servers with a
// local log directory or file specified
func SetupLogTails(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) chan bool {
	stop := make(chan bool)

	for _, server := range servers {
		if server.Config.LogLocation == "" {
			continue
		}

		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		if globalCollectionOpts.DebugLogs || globalCollectionOpts.TestRun {
			prefixedLogger.PrintInfo("Setting up log tail for %s", server.Config.LogLocation)
		}

		logStream := logReceiver(server, globalCollectionOpts, prefixedLogger, nil, stop)
		err := setupLogLocationTail(server.Config.LogLocation, logStream, prefixedLogger, stop)
		if err != nil {
			prefixedLogger.PrintError("ERROR - %s", err)
		}
	}
	return stop
}

// TestLogTail - Tests the tailing of a log file (without watching it continuously)
// as well as parsing and analyzing the log data
func TestLogTail(server state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger) error {
	stop := make(chan bool)

	logLinePrefix, err := getPostgresSetting("log_line_prefix", server, globalCollectionOpts, prefixedLogger)
	if err != nil {
		return err
	} else if !logs.IsSupportedPrefix(logLinePrefix) {
		return fmt.Errorf("Unsupported log_line_prefix setting: '%s'", logLinePrefix)
	}

	logTestSucceeded := make(chan bool, 1)

	logStream := logReceiver(server, globalCollectionOpts, prefixedLogger, logTestSucceeded, stop)
	err = setupLogLocationTail(server.Config.LogLocation, logStream, prefixedLogger, stop)
	if err != nil {
		return err
	}

	db, err := postgres.EstablishConnection(server, prefixedLogger, globalCollectionOpts, "")
	if err == nil {
		db.Exec(postgres.QueryMarkerSQL + fmt.Sprintf("DO $$BEGIN\nRAISE LOG 'pganalyze-collector-identify: %s';\nEND$$;", server.Config.SectionName))
		db.Close()
	}

	select {
	case <-logTestSucceeded:
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("Timeout")
	}
}

func tailFile(path string, out chan<- string, prefixedLogger *util.Logger) (chan bool, error) {
	prefixedLogger.PrintVerbose("Tailing log file %s", path)

	t, err := tail.TailFile(path, tail.Config{Follow: true, MustExist: true, ReOpen: true, Logger: tail.DiscardingLogger})
	if err != nil {
		return nil, fmt.Errorf("Failed to setup log tail: %s", err)
	}

	stop := make(chan bool)

	go func() {
		defer t.Cleanup()
		for {
			select {
			case line := <-t.Lines:
				out <- line.Text
			case <-stop:
				prefixedLogger.PrintVerbose("Stopping log tail for %s (stop requested)", path)
				t.Stop()
				return
			}
		}
	}()

	return stop, nil
}

func isAcceptableLogFile(fileName string, fileNameFilter string) bool {
	if fileNameFilter != "" && fileName != fileNameFilter {
		return false
	}

	if strings.HasSuffix(fileName, ".gz") || strings.HasSuffix(fileName, ".bz2") {
		return false
	}

	return true
}

func filterOutString(strings []string, stringToBeRemoved string) []string {
	newStrings := []string{}
	for _, str := range strings {
		if str != stringToBeRemoved {
			newStrings = append(newStrings, str)
		}
	}
	return newStrings
}

const maxOpenTails = 10

func setupLogLocationTail(logLocation string, out chan<- string, prefixedLogger *util.Logger, stop <-chan bool) error {
	prefixedLogger.PrintVerbose("Searching for log file(s) in %s", logLocation)

	openFiles := make(map[string]chan bool)
	openFilesByAge := []string{}
	fileNameFilter := ""

	statInfo, err := os.Stat(logLocation)
	if err != nil {
		return err
	} else if !statInfo.IsDir() {
		fileNameFilter = logLocation
		logLocation = filepath.Dir(logLocation)
	}

	files, err := ioutil.ReadDir(logLocation)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		// Note that we are sorting descending here, i.e. we want the newest files
		// first
		return files[i].ModTime().After(files[j].ModTime())
	})

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if len(openFiles) >= maxOpenTails {
			break
		}

		fileName := path.Join(logLocation, f.Name())

		if isAcceptableLogFile(fileName, fileNameFilter) {
			var logTailStop chan bool
			logTailStop, err = tailFile(fileName, out, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
			} else {
				openFiles[fileName] = logTailStop
				openFilesByAge = append(openFilesByAge, fileName)
			}
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				//prefixedLogger.PrintVerbose("Received fsnotify event: %s %s", event.Op.String(), event.Name)
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
					_, exists := openFiles[event.Name]
					if isAcceptableLogFile(event.Name, fileNameFilter) && !exists {
						if len(openFiles) >= maxOpenTails {
							var oldestFile string
							oldestFile, openFilesByAge = openFilesByAge[0], openFilesByAge[1:]
							logTailStop, ok := openFiles[oldestFile]
							if ok {
								logTailStop <- true
								delete(openFiles, oldestFile)
							}
						}
						var logTailStop chan bool
						logTailStop, err = tailFile(event.Name, out, prefixedLogger)
						if err != nil {
							prefixedLogger.PrintError("ERROR - %s", err)
						} else {
							openFiles[event.Name] = logTailStop
							openFilesByAge = append(openFilesByAge, event.Name)
						}
					}
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename || event.Op&fsnotify.Chmod == fsnotify.Chmod {
					logTailStop, ok := openFiles[event.Name]
					if ok {
						logTailStop <- true
						delete(openFiles, event.Name)
					}
					openFilesByAge = filterOutString(openFilesByAge, event.Name)
				}
			case err = <-watcher.Errors:
				prefixedLogger.PrintError("ERROR - fsnotify watcher failure: %s", err)
			case <-stop:
				prefixedLogger.PrintVerbose("Log file fsnotify watcher received stop signal")
				for fileName, logTailStop := range openFiles {
					logTailStop <- true
					delete(openFiles, fileName)
				}
				openFilesByAge = []string{}
				return
			}
		}
	}()

	err = watcher.Add(logLocation)
	if err != nil {
		return err
	}

	return nil
}

func logReceiver(server state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger, logTestSucceeded chan<- bool, stop <-chan bool) chan<- string {
	logStream := make(chan string)

	go func() {
		var logLines []state.LogLine

		// Only ingest log lines that were written in the last minute before startup,
		// or later, so we avoid resending full large files on collector restarts
		// TODO: Use prevState here instead to get the last logline we saw
		linesNewerThan := time.Now().Add(-1 * time.Minute)

		// Use a timeout to clear out loglines that don't have any follow-on lines
		// (the threshold used in AnalyzeInGroupsAndSend is 3 seconds)
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(3 * time.Second)
			timeout <- true
		}()

		for {
			select {
			case line, ok := <-logStream:
				if !ok {
					return
				}

				// We ignore failures here since we want the per-backend stitching logic
				// that runs later on (and any other parsing errors will just be ignored)
				logLine, _ := logs.ParseLogLineWithPrefix("", line)
				logLine.CollectedAt = time.Now()
				logLine.UUID = uuid.NewV4()

				// Ignore loglines which are outside our time window
				nullTime := time.Time{}
				if logLine.OccurredAt != nullTime && logLine.OccurredAt.Before(linesNewerThan) {
					continue
				}

				logLines = append(logLines, logLine)
				logLines = logs.AnalyzeInGroupsAndSend(server, logLines, globalCollectionOpts, prefixedLogger, logTestSucceeded)
			case <-timeout:
				if len(logLines) > 0 {
					logLines = logs.AnalyzeInGroupsAndSend(server, logLines, globalCollectionOpts, prefixedLogger, logTestSucceeded)
				}
				go func() {
					time.Sleep(3 * time.Second)
					timeout <- true
				}()
			case <-stop:
				return
			}
		}
	}()

	return logStream
}
