package logs

import (
	"fmt"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	uuid "github.com/satori/go.uuid"
)

func PrintDebugInfo(logFileContents string, logLines []state.LogLine, samples []state.PostgresQuerySample) {
	fmt.Printf("log lines: %d, query samples: %d\n", len(logLines), len(samples))
	groups := map[pganalyze_collector.LogLineInformation_LogClassification]int{}
	unclassifiedLogLines := []state.LogLine{}
	for _, logLine := range logLines {
		if logLine.ParentUUID != uuid.Nil {
			continue
		}

		groups[logLine.Classification]++

		if logLine.Classification == pganalyze_collector.LogLineInformation_UNKNOWN_LOG_CLASSIFICATION {
			unclassifiedLogLines = append(unclassifiedLogLines, logLine)
		}
	}

	for classification, count := range groups {
		fmt.Printf("%d x %s\n", count, classification)
	}

	if len(unclassifiedLogLines) > 0 {
		fmt.Printf("\nUnclassified log lines:\n")
		for i, logLine := range unclassifiedLogLines {
			fullLine := logFileContents[logLine.ByteStart:logLine.ByteEnd]
			if fullLine == "" {
				fullLine = fmt.Sprintf("line #%d at %s", i, logLine.OccurredAt.Format(time.RFC3339Nano))
			}
			lineContent := logFileContents[logLine.ByteContentStart:logLine.ByteEnd]
			if lineContent == "" {
				lineContent = logLine.Content
			}
			fmt.Printf("%s\n", fullLine)
			fmt.Printf("  Level: %s\n", logLine.LogLevel)
			fmt.Printf("  Content: %#v\n", lineContent)
			fmt.Printf("---\n")
		}
	}
}
