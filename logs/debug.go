package logs

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func PrintDebugInfo(logLines []state.LogLine, samples []state.PostgresQuerySample) {
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
		fmt.Printf("%d x %s (%d)\n", count, classification, classification)
	}

	if len(unclassifiedLogLines) > 0 {
		fmt.Printf("\nUnclassified log lines:\n")
		for _, logLine := range unclassifiedLogLines {
			fmt.Printf("  %s: %s", logLine.LogLevel, logLine.Content)
			fmt.Printf("---\n")
		}
	}
}

func PrintDebugLogLines(logLines []state.LogLine, classifications map[pganalyze_collector.LogLineInformation_LogClassification]bool) {
	fmt.Println("\nParsed log lines:")
	linesById := make(map[uuid.UUID]*state.LogLine)
	for _, logLine := range logLines {
		logLine := logLine
		linesById[logLine.UUID] = &logLine
		if len(classifications) > 0 {
			var classifiedLine *state.LogLine
			if logLine.ParentUUID == uuid.Nil {
				classifiedLine = &logLine
			} else {
				classifiedLine = linesById[logLine.ParentUUID]
			}
			if _, ok := classifications[classifiedLine.Classification]; !ok {
				continue
			}
		}
		detailsStr, err := json.Marshal(logLine.Details)
		if err != nil {
			panic(err)
		}
		fmt.Printf("  %s: %s", logLine.LogLevel, logLine.Content)
		if logLine.ParentUUID == uuid.Nil {
			fmt.Printf("  Classification: %s (%d)\n", logLine.Classification, logLine.Classification)
		}
		if len(logLine.Details) > 0 {
			fmt.Printf("  Details:        %s\n", detailsStr)
		}
		fmt.Printf("---\n")
	}
}

// ParseLogClassifications parses a comma-separated list of log classification
// numbers, or "all", for PrintDebugLogLines. "all" returns an empty map,
// which PrintDebugLogLines treats as every classification.
func ParseLogClassifications(list string) (map[pganalyze_collector.LogLineInformation_LogClassification]bool, error) {
	classifications := make(map[pganalyze_collector.LogLineInformation_LogClassification]bool)
	if list == "all" {
		return classifications, nil
	}
	for _, entry := range strings.Split(list, ",") {
		if entry == "all" {
			return nil, errors.New("\"all\" can't be combined with other classifications")
		}
		value, err := strconv.ParseInt(entry, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid classification %q: must be an integer or \"all\"", entry)
		}
		if _, ok := pganalyze_collector.LogLineInformation_LogClassification_name[int32(value)]; !ok {
			return nil, fmt.Errorf("unknown classification %d", value)
		}
		classifications[pganalyze_collector.LogLineInformation_LogClassification(value)] = true
	}
	return classifications, nil
}

var HerokuPostgresDebugRegexp = regexp.MustCompile(`^(\w+ \d+ \d+:\d+:\d+ \w+ app\[postgres\] \w+ )?\[(\w+)\] \[\d+-\d+\] (.+)`)

type MaybeHerokuLogReader struct {
	LineReader
}

func NewMaybeHerokuLogReader(r io.Reader) *MaybeHerokuLogReader {
	return &MaybeHerokuLogReader{bufio.NewReader((r))}
}

func (lr *MaybeHerokuLogReader) ReadString(delim byte) (string, error) {
	line, err := lr.LineReader.ReadString(delim)
	if err != nil {
		return "", err
	}
	contentParts := HerokuPostgresDebugRegexp.FindStringSubmatch(line)
	if len(contentParts) == 4 {
		return contentParts[3], nil
	}

	return line, nil
}
