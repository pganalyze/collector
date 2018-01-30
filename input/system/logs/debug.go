package logs

import (
  "fmt"
	"github.com/pganalyze/collector/state"
  "github.com/pganalyze/collector/output/pganalyze_collector"
)

func PrintDebugInfo(logLines []state.LogLine, samples []state.PostgresQuerySample) {
  fmt.Printf("log lines: %d, query samples: %d\n", len(logLines), len(samples))
  groups := map[pganalyze_collector.LogLineInformation_LogClassification]int{}
  for _, logLine := range logLines {
    groups[logLine.Classification] += 1
  }
  for classification, count := range groups {
    fmt.Printf("%d x %s\n", count, classification)
  }
}
