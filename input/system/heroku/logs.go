package heroku

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bmizerany/lpx"
	"github.com/kr/logfmt"
	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type HerokuLogStreamItem struct {
	Header  lpx.Header
	Content []byte
	Path    string
}

type SystemSample struct {
	Source            string  `logfmt:"source"`
	LoadAvg1min       float64 `logfmt:"sample#load-avg-1m"`
	LoadAvg5min       float64 `logfmt:"sample#load-avg-5m"`
	LoadAvg15min      float64 `logfmt:"sample#load-avg-15m"`
	MemoryPostgresKb  string  `logfmt:"sample#memory-postgres"`
	MemoryTotalUsedKb string  `logfmt:"sample#memory-total"`
	MemoryFreeKb      string  `logfmt:"sample#memory-free"`
	MemoryCachedKb    string  `logfmt:"sample#memory-cached"`
	StorageBytesUsed  string  `logfmt:"sample#db_size"`
	ReadIops          float64 `logfmt:"sample#read-iops"`
	WriteIops         float64 `logfmt:"sample#write-iops"`
}

func catchIdentifyServerLine(sourceName string, content string, sourceToServer map[string]*state.Server, servers []*state.Server) map[string]*state.Server {
	identifyParts := regexp.MustCompile(`^pganalyze-collector-identify: ([\w_]+)`).FindStringSubmatch(content)
	if len(identifyParts) == 2 {
		for _, server := range servers {
			if server.Config.SectionName == identifyParts[1] {
				sourceToServer[sourceName] = server
			}
		}
	}

	return sourceToServer
}

func processSystemMetrics(timestamp time.Time, content []byte, sourceToServer map[string]*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger, namespace string) {
	var sample SystemSample
	err := logfmt.Unmarshal(content, &sample)
	if err != nil {
		logger.PrintError("Failed to unmarshal message: %s\n  %s", err, content)
		return
	}
	sourceName := sample.Source
	if !strings.HasPrefix(sourceName, "HEROKU_POSTGRESQL_") {
		sourceName = "HEROKU_POSTGRESQL_" + sourceName
	}
	server, exists := sourceToServer[namespace+" / "+sourceName]
	if !exists {
		logger.PrintInfo("Ignoring system data since server can't be matched yet - if this keeps showing up you have a configuration error for %s", namespace+" / "+sourceName)
		return
	}
	server.CollectionStatusMutex.Lock()
	if server.CollectionStatus.CollectionDisabled {
		server.CollectionStatusMutex.Unlock()
		return
	}
	server.CollectionStatusMutex.Unlock()

	prefixedLogger := logger.WithPrefix(server.Config.SectionName)

	grant, err := grant.GetDefaultGrant(server, globalCollectionOpts, prefixedLogger)
	if err != nil {
		prefixedLogger.PrintError("Could not get default grant for system snapshot: %s", err)
		return
	}

	system := state.SystemState{}
	system.Info.Type = state.HerokuSystem
	system.Info.SystemID = server.Config.SystemID
	system.Info.SystemScope = server.Config.SystemScope
	system.Scheduler = state.Scheduler{Loadavg1min: sample.LoadAvg1min, Loadavg5min: sample.LoadAvg5min, Loadavg15min: sample.LoadAvg15min}

	memoryPostgresKb, _ := strconv.ParseInt(strings.TrimSuffix(sample.MemoryPostgresKb, "kB"), 10, 64)
	memoryTotalUsedKb, _ := strconv.ParseInt(strings.TrimSuffix(sample.MemoryTotalUsedKb, "kB"), 10, 64)
	memoryFreeKb, _ := strconv.ParseInt(strings.TrimSuffix(sample.MemoryFreeKb, "kB"), 10, 64)
	memoryCachedKb, _ := strconv.ParseInt(strings.TrimSuffix(sample.MemoryCachedKb, "kB"), 10, 64)

	system.Memory = state.Memory{
		ApplicationBytes: uint64(memoryPostgresKb * 1024),
		TotalBytes:       uint64(memoryTotalUsedKb * 1024),
		FreeBytes:        uint64(memoryFreeKb * 1024),
		CachedBytes:      uint64(memoryCachedKb * 1024),
	}

	system.Disks = make(state.DiskMap)
	system.Disks["default"] = state.Disk{}

	storageBytesUsed, _ := strconv.ParseUint(strings.TrimSuffix(sample.StorageBytesUsed, "bytes"), 10, 64)
	system.DiskPartitions = make(state.DiskPartitionMap)
	system.DiskPartitions["/"] = state.DiskPartition{
		DiskName:  "default",
		UsedBytes: storageBytesUsed,
	}

	system.DiskStats = make(state.DiskStatsMap)
	system.DiskStats["default"] = state.DiskStats{
		DiffedOnInput: true,
		DiffedValues: &state.DiffedDiskStats{
			ReadOperationsPerSecond:  sample.ReadIops,
			WriteOperationsPerSecond: sample.WriteIops,
		},
	}

	err = output.SubmitCompactSystemSnapshot(server, grant, globalCollectionOpts, prefixedLogger, system, timestamp)
	if err != nil {
		prefixedLogger.PrintError("Failed to upload/send compact system snapshot: %s", err)
		return
	}

	return
}

func logStreamItemToLogLine(item HerokuLogStreamItem, servers []*state.Server, sourceToServer map[string]*state.Server, now time.Time, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (map[string]*state.Server, *state.LogLine, string) {
	timestamp, err := time.Parse(time.RFC3339, string(item.Header.Time))
	if err != nil {
		return sourceToServer, nil, ""
	}

	if string(item.Header.Name) != "app" {
		return sourceToServer, nil, ""
	}

	namespace := "default"
	if strings.HasPrefix(item.Path, "/logs/") {
		namespace = strings.Replace(item.Path, "/logs/", "", 1)
	}

	if string(item.Header.Procid) == "heroku-postgres" {
		processSystemMetrics(timestamp, item.Content, sourceToServer, globalCollectionOpts, logger, namespace)
		return sourceToServer, nil, ""
	}

	parts := regexp.MustCompile(`^postgres.(\d+)`).FindStringSubmatch(string(item.Header.Procid))
	if len(parts) != 2 {
		return sourceToServer, nil, ""
	}
	backendPid, _ := strconv.ParseInt(parts[1], 10, 32)

	lineParts := regexp.MustCompile(`^\[(\w+)\] \[(\d+)-(\d+)\] (.+)`).FindStringSubmatch(string(item.Content))
	if len(lineParts) != 5 {
		fmt.Printf("ERR: %s\n", string(item.Content))
		return sourceToServer, nil, ""
	}

	sourceName := lineParts[1]
	if !strings.HasPrefix(sourceName, "HEROKU_POSTGRESQL_") {
		sourceName = "HEROKU_POSTGRESQL_" + sourceName
	}
	sourceName = namespace + " / " + sourceName
	logLineNumber, _ := strconv.ParseInt(lineParts[2], 10, 32)
	logLineNumberChunk, _ := strconv.ParseInt(lineParts[3], 10, 32)
	prefixedContent := lineParts[4]

	logLine, _ := logs.ParseLogLineWithPrefix("", prefixedContent+"\n")

	sourceToServer = catchIdentifyServerLine(sourceName, logLine.Content, sourceToServer, servers)

	logLine.OccurredAt = timestamp
	logLine.BackendPid = int32(backendPid)
	logLine.LogLineNumber = int32(logLineNumber)
	logLine.LogLineNumberChunk = int32(logLineNumberChunk)

	return sourceToServer, &logLine, sourceName
}

func setupLogTransformer(ctx context.Context, wg *sync.WaitGroup, servers []*state.Server, in <-chan HerokuLogStreamItem, out chan state.ParsedLogStreamItem, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		sourceToServer := make(map[string]*state.Server)

		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-in:
				if !ok {
					return
				}

				fmt.Printf("log drain received: %s\n", string(item.Content))

				now := time.Now()

				var logLine *state.LogLine
				var sourceName string
				sourceToServer, logLine, sourceName = logStreamItemToLogLine(item, servers, sourceToServer, now, globalCollectionOpts, logger)
				if logLine == nil || sourceName == "" {
					continue
				}

				server, exists := sourceToServer[sourceName]
				if !exists {
					logger.PrintInfo("Ignoring log line since server can't be matched yet - if this keeps showing up you have a configuration error for %s", sourceName)
					continue
				}

				logLine.Username = server.Config.GetDbUsername()
				logLine.Database = server.Config.GetDbName()
				out <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: *logLine}
			}
		}
	}()
}
