package logs

import (
	"fmt"

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
		for _, logLine := range unclassifiedLogLines {
			fmt.Printf("%s\n", logFileContents[logLine.ByteStart:logLine.ByteEnd+1])
			fmt.Printf("  Level: %s\n", logLine.LogLevel)
			fmt.Printf("  Content: %#v\n", logFileContents[logLine.ByteContentStart:logLine.ByteEnd+1])
			fmt.Printf("---\n")
		}
	}
}
