package selfhosted

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/papertrail/go-tail/follower"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/logs/stream"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

const settingValueSQL string = `
SELECT setting
	FROM pg_settings
 WHERE name = '%s'`

func getPostgresSetting(settingName string, server *state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger) (string, error) {
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
func DiscoverLogLocation(servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
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
			logDirectoryBytes, err := exec.Command("/usr/bin/pganalyze-collector-helper", "log_directory").Output()
			if err != nil {
				prefixedLogger.PrintError("ERROR - Could not run helper process: %s", err)
				continue
			}
			logDirectory := string(logDirectoryBytes)
			if logDirectory == "" {
				prefixedLogger.PrintError("ERROR - Could not retrieve log_directory setting from Postgres")
				continue
			}
			if !strings.HasPrefix(logDirectory, "/") {
				logDirectory = status.DataDirectory + "/" + logDirectory
			}
			prefixedLogger.PrintInfo("Found log location, add this to your pganalyze-collector.conf in the [%s] section:\ndb_log_location = %s", server.Config.SectionName, logDirectory)
		} else { // assume stdout/stderr redirect to logfile, typical with postgresql-common on Ubuntu/Debian
			prefixedLogger.PrintInfo("Discovering log directory using open files in postmaster (PID %d)...", status.PostmasterPid)
			logFile, err := filepath.EvalSymlinks("/proc/" + strconv.FormatInt(int64(status.PostmasterPid), 10) + "/fd/1")
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
				continue
			}
			prefixedLogger.PrintInfo("Found log location, add this to your pganalyze-collector.conf in the [%s] section:\ndb_log_location = %s", server.Config.SectionName, logFile)
		}
	}
}

// SetupLogTails - Sets up continuously running log tails for all servers with a
// local log directory or file specified
func SetupLogTails(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		if server.Config.LogLocation != "" {
			if globalCollectionOpts.DebugLogs || globalCollectionOpts.TestRun {
				prefixedLogger.PrintInfo("Setting up log tail for %s", server.Config.LogLocation)
			}

			logStream := logReceiver(ctx, server, globalCollectionOpts, prefixedLogger, nil)
			err := setupLogLocationTail(ctx, server.Config.LogLocation, logStream, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
			}
		} else if server.Config.LogDockerTail != "" {
			if globalCollectionOpts.DebugLogs || globalCollectionOpts.TestRun {
				prefixedLogger.PrintInfo("Setting up docker logs tail for %s", server.Config.LogDockerTail)
			}

			logStream := logReceiver(ctx, server, globalCollectionOpts, prefixedLogger, nil)
			err := setupDockerTail(ctx, server.Config.LogDockerTail, logStream, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
			}
		}
	}
}

func tailFile(ctx context.Context, path string, out chan<- string, prefixedLogger *util.Logger) error {
	prefixedLogger.PrintVerbose("Tailing log file %s", path)

	t, err := follower.New(path, follower.Config{
		Whence: io.SeekEnd,
		Offset: 0,
		Reopen: true,
	})
	if err != nil {
		return fmt.Errorf("Failed to setup log tail: %s", err)
	}

	go func() {
		defer t.Close()
		for {
			select {
			case line := <-t.Lines():
				out <- line.String()
			case <-ctx.Done():
				prefixedLogger.PrintVerbose("Stopping log tail for %s (stop requested)", path)
				t.Close()
				return
			}
		}
		if t.Err() != nil {
			t.Close()
			prefixedLogger.PrintError("Failed log file tail: %s", t.Err())
		}
	}()

	return nil
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

func setupLogLocationTail(ctx context.Context, logLocation string, out chan<- string, prefixedLogger *util.Logger) error {
	prefixedLogger.PrintVerbose("Searching for log file(s) in %s", logLocation)

	openFiles := make(map[string]context.CancelFunc)
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
			tailCtx, tailCancel := context.WithCancel(ctx)
			err = tailFile(tailCtx, fileName, out, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
			} else {
				openFiles[fileName] = tailCancel
				openFilesByAge = append(openFilesByAge, fileName)
			}
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify new: %s", err)
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
							tailCancel, ok := openFiles[oldestFile]
							if ok {
								tailCancel()
								delete(openFiles, oldestFile)
							}
						}
						tailCtx, tailCancel := context.WithCancel(ctx)
						err = tailFile(tailCtx, event.Name, out, prefixedLogger)
						if err != nil {
							tailCancel()
							prefixedLogger.PrintError("ERROR - %s", err)
						} else {
							openFiles[event.Name] = tailCancel
							openFilesByAge = append(openFilesByAge, event.Name)
						}
					}
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename || event.Op&fsnotify.Chmod == fsnotify.Chmod {
					tailCancel, ok := openFiles[event.Name]
					if ok {
						tailCancel()
						delete(openFiles, event.Name)
					}
					openFilesByAge = filterOutString(openFilesByAge, event.Name)
				}
			case err = <-watcher.Errors:
				prefixedLogger.PrintError("ERROR - fsnotify watcher failure: %s", err)
			case <-ctx.Done():
				prefixedLogger.PrintVerbose("Log file fsnotify watcher received stop signal")
				for fileName, tailCancel := range openFiles {
					// TODO: This cancel might actually not be necessary since we are
					// already canceling the parent context?
					tailCancel()
					delete(openFiles, fileName)
				}
				openFilesByAge = []string{}
				return
			}
		}
	}()

	err = watcher.Add(logLocation)
	if err != nil {
		return fmt.Errorf("fsnotify add \"%s\": %s", logLocation, err)
	}

	return nil
}

func setupDockerTail(ctx context.Context, containerName string, out chan<- string, prefixedLogger *util.Logger) error {
	var err error

	cmd := exec.Command("docker", "logs", containerName, "-f", "--tail", "0")
	stderr, _ := cmd.StderrPipe()

	scanner := bufio.NewScanner(stderr)
	go func() {
		for scanner.Scan() {
			out <- scanner.Text()
		}
	}()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Error starting docker log tail: %s", err)
	}

	go func() {
		defer cmd.Wait()
		for {
			select {
			case <-ctx.Done():
				prefixedLogger.PrintVerbose("Docker log tail received stop signal")
				if err := cmd.Process.Kill(); err != nil {
					prefixedLogger.PrintError("Failed to kill docker log tail process when stop received: %s", err)
				}
				return
			}
		}
	}()

	return nil
}

func logReceiver(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger, logTestSucceeded chan<- bool) chan<- string {
	logStream := make(chan string)

	go func() {
		var logLines []state.LogLine

		// Only ingest log lines that were written in the last minute before startup,
		// or later, so we avoid resending full large files on collector restarts
		// TODO: Use prevState here instead to get the last logline we saw
		linesNewerThan := time.Now().Add(-1 * time.Minute)

		// Use a timeout to clear out loglines that don't have any follow-on lines
		// (the threshold used in logs.ProcessLogStream is 3 seconds)
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
				logLines = stream.ProcessLogStream(server, logLines, globalCollectionOpts, prefixedLogger, logTestSucceeded, stream.LogTestCollectorIdentify)
			case <-timeout:
				if len(logLines) > 0 {
					logLines = stream.ProcessLogStream(server, logLines, globalCollectionOpts, prefixedLogger, logTestSucceeded, stream.LogTestCollectorIdentify)
				}
				go func() {
					time.Sleep(3 * time.Second)
					timeout <- true
				}()
			case <-ctx.Done():
				return
			}
		}
	}()

	return logStream
}
