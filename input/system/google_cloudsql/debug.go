package google_cloudsql

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/logs/stream"
	"github.com/pganalyze/collector/state"
	uuid "github.com/satori/go.uuid"
)

func DebugParseAndAnalyzeBufferGcp(buffer string) (state.LogFile, []state.PostgresQuerySample) {
	var lines []googleLogMessage
	err := json.Unmarshal([]byte(buffer), &lines)
	if err != nil {
		fmt.Printf("ERROR: Log file is not valid JSON: %s", err)
	}

	now := time.Now()

	var logLines []state.LogLine
	for i, line := range lines {
		content := line.TextPayload
		logLine, ok := logs.ParseLogLineWithPrefix("", content+"\n")
		if !ok {
			fmt.Printf("WARNING: could not parse item %d at %s: %s", i, line.Timestamp, content)
			continue
		}
		logLine.CollectedAt = now
		ts, err := time.Parse(time.RFC3339Nano, line.Timestamp)
		if err != nil {
			fmt.Printf("WARNING: could not parse item %d timestamp: %s", i, line.Timestamp)
			continue
		}

		logLine.OccurredAt = ts
		logLine.UUID = uuid.NewV4()

		logLines = append(logLines, logLine)
	}

	logState, logFile, _, err := stream.AnalyzeStreamInGroups(logLines, now.Add(stream.StreamReadyThreshold).Add(time.Second*1))
	if err != nil {
		fmt.Printf("ERROR: Failed log analysis: %s", err)
	}
	return logFile, logState.QuerySamples
}
