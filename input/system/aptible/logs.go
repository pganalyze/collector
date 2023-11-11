package aptible

import (
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type AptibleLog struct {
	Time     string `json:"time"`
	Source   string `json:"source"`
	Database string `json:"database"`
	Offset   uint64 `json:"offset"`
	Log      string `json:"log"`
}

// Maybe useful
func findServerByIdentifier(servers []*state.Server, identifier config.ServerIdentifier) *state.Server {
	for _, s := range servers {
		if s.Config.Identifier == identifier {
			return s
		}
	}
	return nil
}

func HandleLogMessage(logMessage *AptibleLog, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) {

	if logMessage.Source != "database" || logMessage.Database != "healthie-staging-14" {
		return
	}

	for _, server := range servers {
		if server.Config.SectionName == "healthie-staging-14" {
			prefixedLogger := logger.WithPrefix(server.Config.SectionName)
			logLine, ok := logs.ParseLogLineWithPrefix(logs.LogPrefixCustom3, logMessage.Log+"\n", nil)
			if ok {
				occurredAt, err := time.Parse(time.RFC3339, logMessage.Time)
				if err != nil {
					prefixedLogger.Destination.Fatalf("Error happened time parsing. Err: %s\n", err)
				}
				prefixedLogger.PrintVerbose("Submitting log message")
				logLine.OccurredAt = occurredAt
				parsedLogStream <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
			}
		}
	}
}
