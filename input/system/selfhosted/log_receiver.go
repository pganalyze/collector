package selfhosted

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/pganalyze/collector/input/system/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

func SetupLogTails(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, server := range servers {
		if server.Config.LogLocation == "" {
			continue
		}

		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		if globalCollectionOpts.DebugLogs {
			prefixedLogger.PrintInfo("Setting up log tail for %s", server.Config.LogLocation)
		}

		logStream := logReceiver(server, globalCollectionOpts, prefixedLogger)
		setupLogLocationTail(server.Config.LogLocation, logStream, prefixedLogger)
	}
}

func tailFile(path string, out chan<- string, prefixedLogger *util.Logger) {
	t, err := tail.TailFile(path, tail.Config{Follow: true, MustExist: true, ReOpen: true, Logger: tail.DiscardingLogger})
	if err != nil {
		prefixedLogger.PrintError("Error: %s", err)
		return
	}
	defer t.Cleanup()
	for line := range t.Lines {
		out <- line.Text
	}
}

func setupLogLocationTail(logLocation string, out chan<- string, prefixedLogger *util.Logger) {
	statInfo, err := os.Stat(logLocation)
	if err != nil {
		prefixedLogger.PrintError("Error: %s", err)
		return
	}

	if !statInfo.IsDir() {
		go tailFile(logLocation, out, prefixedLogger)
		return
	}

	files, err := ioutil.ReadDir(logLocation)
	if err != nil {
		prefixedLogger.PrintError("Error: %s", err)
		return
	}

	for _, f := range files {
		if !f.IsDir() {
			go tailFile(path.Join(logLocation, f.Name()), out, prefixedLogger)
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		prefixedLogger.PrintError("Error: %s", err)
		return
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					go tailFile(event.Name, out, prefixedLogger)
				}
			case err := <-watcher.Errors:
				prefixedLogger.PrintError("Error: %s", err)
			}
		}
	}()

	err = watcher.Add(logLocation)
	if err != nil {
		prefixedLogger.PrintError("Error: %s", err)
		return
	}

	return
}

func logReceiver(server state.Server, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger) chan<- string {
	logStream := make(chan string)

	go func() {
		var logLines []state.LogLine

		// Only ingest log lines that were written in the last minute before startup,
		// or later, so we avoid resending full large files on colletor restarts
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
				logLines = logs.AnalyzeInGroupsAndSend(server, logLines, globalCollectionOpts, prefixedLogger)
			case <-timeout:
				if len(logLines) > 0 {
					logLines = logs.AnalyzeInGroupsAndSend(server, logLines, globalCollectionOpts, prefixedLogger)
				}
				go func() {
					time.Sleep(3 * time.Second)
					timeout <- true
				}()
			}
		}
	}()

	return logStream
}
