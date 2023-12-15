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
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/papertrail/go-tail/follower"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

type SelfHostedLogStreamItem struct {
	Line string

	// Optional, only used for syslog messages
	OccurredAt         time.Time
	BackendPid         int32
	LogLineNumber      int32
	LogLineNumberChunk int32
}

const settingValueSQL string = `
SELECT setting
	FROM pg_settings
 WHERE name = '%s'`

func getPostgresSetting(ctx context.Context, settingName string, server *state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger) (string, error) {
	var value string

	db, err := postgres.EstablishConnection(ctx, server, prefixedLogger, globalCollectionOpts, "")
	if err != nil {
		return "", fmt.Errorf("Could not connect to database to retrieve \"%s\": %s", settingName, err)
	}

	err = db.QueryRowContext(ctx, postgres.QueryMarkerSQL+fmt.Sprintf(settingValueSQL, settingName)).Scan(&value)
	db.Close()
	if err != nil {
		return "", fmt.Errorf("Could not read \"%s\" setting: %s", settingName, err)
	}

	return value, nil
}

// DiscoverLogLocation - Tries to find the log location for a currently running Postgres
// process and outputs the presumed location using the logger
func DiscoverLogLocation(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		if server.Config.DbHost != "localhost" && server.Config.DbHost != "127.0.0.1" {
			prefixedLogger.PrintWarning("WARNING - Database hostname is not localhost - Log Insights requires the collector to run on the database server directly for self-hosted systems")
		}

		logDestination, err := getPostgresSetting(ctx, "log_destination", server, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("ERROR - %s", err)
			continue
		}

		if logDestination == "syslog" {
			prefixedLogger.PrintInfo("WARNING: Logging via syslog - please check our setup guide for rsyslogd or syslog-ng instructions")
			continue
		} else if logDestination != "stderr" {
			prefixedLogger.PrintError("ERROR - Unsupported log_destination \"%s\"", logDestination)
			continue
		}

		loggingCollector, err := getPostgresSetting(ctx, "logging_collector", server, globalCollectionOpts, prefixedLogger)
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
			logDirectory, err := getPostgresSetting(ctx, "log_directory", server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - Could not retrieve log_directory setting from Postgres: %s", err)
				continue
			}

			if strings.HasPrefix(logDirectory, "/") {
				prefixedLogger.PrintInfo("Found log location, add this to your pganalyze-collector.conf in the [%s] section:\ndb_log_location = %s", server.Config.SectionName, logDirectory)
			} else {
				prefixedLogger.PrintInfo("WARNING: Found relative log location \"%s\" inside data directory - please check our setup guide for instructions\n", logDirectory)
			}
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

func SetupLogTailForServer(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, server *state.Server, parsedLogStream chan state.ParsedLogStreamItem) error {
	if globalCollectionOpts.DebugLogs || globalCollectionOpts.TestRun {
		logger.PrintInfo("Setting up log tail for %s", server.Config.LogLocation)
	}

	logStream := setupLogTransformer(ctx, wg, server, globalCollectionOpts, logger, parsedLogStream)
	return setupLogLocationTail(ctx, server.Config.LogLocation, logStream, logger)
}

// SetupLogTails - Sets up continuously running log tails for all servers with a
// local log directory or file specified
func SetupLogTails(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {
	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		if server.Config.LogLocation != "" {
			err := SetupLogTailForServer(ctx, wg, globalCollectionOpts, logger, server, parsedLogStream)
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
			}
		} else if server.Config.LogDockerTail != "" {
			if globalCollectionOpts.DebugLogs || globalCollectionOpts.TestRun {
				prefixedLogger.PrintInfo("Setting up docker logs tail for %s", server.Config.LogDockerTail)
			}

			logStream := setupLogTransformer(ctx, wg, server, globalCollectionOpts, prefixedLogger, parsedLogStream)
			err := setupDockerTail(ctx, server.Config.LogDockerTail, logStream, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
			}
		} else if server.Config.LogSyslogServer != "" {
			logStream := setupLogTransformer(ctx, wg, server, globalCollectionOpts, prefixedLogger, parsedLogStream)
			err := setupSyslogHandler(ctx, server.Config, logStream, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("ERROR - %s", err)
			}
		}
	}
}

func tailFile(ctx context.Context, path string, out chan<- SelfHostedLogStreamItem, prefixedLogger *util.Logger) error {
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
	TailLoop:
		for {
			select {
			case line := <-t.Lines():
				out <- SelfHostedLogStreamItem{Line: line.String()}
			case <-ctx.Done():
				prefixedLogger.PrintVerbose("Stopping log tail for %s (stop requested)", path)
				break TailLoop
			}
		}
		if t.Err() != nil {
			prefixedLogger.PrintError("Failed log file tail: %s", t.Err())
		}
	}()

	return nil
}

func isAcceptableLogFile(fileName string, fileNameFilter string) bool {
	if fileNameFilter != "" && fileName != fileNameFilter {
		return false
	}

	if strings.HasSuffix(fileName, ".gz") || strings.HasSuffix(fileName, ".bz2") || strings.HasSuffix(fileName, ".csv") {
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

func setupLogLocationTail(ctx context.Context, logLocation string, out chan<- SelfHostedLogStreamItem, prefixedLogger *util.Logger) error {
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
				tailCancel()
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

func setupDockerTail(ctx context.Context, containerName string, out chan<- SelfHostedLogStreamItem, prefixedLogger *util.Logger) error {
	var err error

	cmd := exec.Command("docker", "logs", "-f", "--tail", "0", containerName)
	stderr, _ := cmd.StderrPipe()

	scanner := bufio.NewScanner(stderr)
	go func() {
		for scanner.Scan() {
			out <- SelfHostedLogStreamItem{Line: scanner.Text()}
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

func setupLogTransformer(ctx context.Context, wg *sync.WaitGroup, server *state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger, parsedLogStream chan state.ParsedLogStreamItem) chan<- SelfHostedLogStreamItem {
	logStream := make(chan SelfHostedLogStreamItem)
	tz := server.GetLogTimezone()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Only ingest log lines that were written in the last minute before startup,
		// or later, so we avoid resending full large files on collector restarts
		// TODO: Use prevState here instead to get the last logline we saw
		linesNewerThan := time.Now().Add(-1 * time.Minute)

		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-logStream:
				if !ok {
					return
				}

				// We ignore failures here since we want the per-backend stitching logic
				// that runs later on (and any other parsing errors will just be ignored)
				// Note that we need to restore the original trailing newlines since
				// AnalyzeStreamInGroups expects them and they are not present in the tail
				// log stream.
				logLine, _ := logs.ParseLogLineWithPrefix("", item.Line+"\n", tz)
				logLine.CollectedAt = time.Now()
				logLine.UUID = uuid.NewV4()

				if logLine.OccurredAt.IsZero() && !item.OccurredAt.IsZero() {
					logLine.OccurredAt = item.OccurredAt
				}
				if logLine.BackendPid == 0 && item.BackendPid != 0 {
					logLine.BackendPid = item.BackendPid
				}
				if logLine.LogLineNumber == 0 && item.LogLineNumber != 0 {
					logLine.LogLineNumber = item.LogLineNumber
				}
				if item.LogLineNumberChunk != 0 {
					logLine.LogLineNumberChunk = item.LogLineNumberChunk
				}

				// Ignore loglines which are outside our time window
				if !logLine.OccurredAt.IsZero() && logLine.OccurredAt.Before(linesNewerThan) {
					continue
				}

				parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
			}
		}
	}()

	return logStream
}
