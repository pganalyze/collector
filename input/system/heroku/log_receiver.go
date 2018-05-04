package heroku

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kr/logfmt"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system/logs"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

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

func SetupLogReceiver(conf config.Config, servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	go logReceiver(servers, conf.HerokuLogStream, globalCollectionOpts, logger)

	for _, server := range servers {
		db, err := postgres.EstablishConnection(server, logger, globalCollectionOpts, "")
		if err == nil {
			db.Exec(postgres.QueryMarkerSQL + fmt.Sprintf("DO $$BEGIN\nRAISE LOG 'pganalyze-collector-identify: %s';\nEND$$;", server.Config.SectionName))
			db.Close()
		}
	}
}

func catchIdentifyServerLine(sourceName string, content string, nameToServer map[string]state.Server, servers []state.Server) map[string]state.Server {
	identifyParts := regexp.MustCompile(`^pganalyze-collector-identify: ([\w_]+)`).FindStringSubmatch(content)
	if len(identifyParts) == 2 {
		for _, server := range servers {
			if server.Config.SectionName == identifyParts[1] {
				nameToServer[sourceName] = server
			}
		}
	}

	return nameToServer
}

func processSystemMetrics(timestamp time.Time, content []byte, nameToServer map[string]state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger, namespace string) {
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
	server, exists := nameToServer[namespace+" / "+sourceName]
	if !exists {
		logger.PrintInfo("Ignoring system data since server can't be matched yet - if this keeps showing up you have a configuration error for %s", namespace+" / "+sourceName)
		return
	}

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
		TotalBytes:       uint64(memoryFreeKb*1024) + uint64(memoryTotalUsedKb*1024),
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

func processLogLine(timestamp time.Time, backendPid int64, logLevel string, content string, nameToServer map[string]state.Server) *state.LogLine {
	var logLine state.LogLine

	logLine.CollectedAt = time.Now()
	logLine.OccurredAt = timestamp
	logLine.BackendPid = int32(backendPid)
	logLine.Content = content
	logLine.UUID = uuid.NewV4()

	if logLevel != "" { // Append-lines don't have a log level
		logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[logLevel])
	}

	return &logLine
}

func processItem(item config.HerokuLogStreamItem, servers []state.Server, nameToServer map[string]state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (map[string]state.Server, *state.LogLine, string) {
	timestamp, err := time.Parse(time.RFC3339, string(item.Header.Time))
	if err != nil {
		return nameToServer, nil, ""
	}

	if string(item.Header.Name) != "app" {
		return nameToServer, nil, ""
	}

	if string(item.Header.Procid) == "heroku-postgres" {
		processSystemMetrics(timestamp, item.Content, nameToServer, globalCollectionOpts, logger, item.Namespace)
		return nameToServer, nil, ""
	}

	parts := regexp.MustCompile(`^postgres.(\d+)`).FindStringSubmatch(string(item.Header.Procid))
	if len(parts) != 2 {
		return nameToServer, nil, ""
	}
	contentParts := regexp.MustCompile(`^\[(\w+)\] \[\d+-\d+\] ( sql_error_code = \w+ (\w+):  )?(.+)`).FindStringSubmatch(string(item.Content))
	if len(contentParts) != 5 {
		fmt.Printf("ERR: %s\n", string(item.Content))
		return nameToServer, nil, ""
	}

	sourceName := contentParts[1]
	if !strings.HasPrefix(sourceName, "HEROKU_POSTGRESQL_") {
		sourceName = "HEROKU_POSTGRESQL_" + sourceName
	}

	nameToServer = catchIdentifyServerLine(item.Namespace+" / "+sourceName, contentParts[4], nameToServer, servers)

	backendPid, _ := strconv.ParseInt(parts[1], 10, 32)
	newLogLine := processLogLine(timestamp, backendPid, contentParts[3], contentParts[4], nameToServer)

	return nameToServer, newLogLine, item.Namespace + " / " + sourceName
}

func logReceiver(servers []state.Server, in <-chan config.HerokuLogStreamItem, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	var logLinesByName map[string][]state.LogLine
	var nameToServer map[string]state.Server

	logLinesByName = make(map[string][]state.LogLine)
	nameToServer = make(map[string]state.Server)

	for {
		item, ok := <-in
		if !ok {
			return
		}

		var newLogLine *state.LogLine
		var sourceName string
		nameToServer, newLogLine, sourceName = processItem(item, servers, nameToServer, globalCollectionOpts, logger)
		if newLogLine != nil && sourceName != "" {
			logLinesByName[sourceName] = append(logLinesByName[sourceName], *newLogLine)
		}

		for sourceName, logLines := range logLinesByName {
			server, exists := nameToServer[sourceName]
			if !exists {
				logger.PrintInfo("Ignoring log line since server can't be matched yet - if this keeps showing up you have a configuration error for %s", sourceName)
				logLinesByName[sourceName] = []state.LogLine{}
				continue
			}

			for idx, logLine := range logLines {
				logLine.Username = server.Config.GetDbUsername()
				logLine.Database = server.Config.GetDbName()
				logLines[idx] = logLine
			}

			prefixedLogger := logger.WithPrefix(server.Config.SectionName)
			logLinesByName[sourceName] = logs.AnalyzeInGroupsAndSend(server, logLines, globalCollectionOpts, prefixedLogger, nil)
		}
	}
}
