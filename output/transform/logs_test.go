package transform_test

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// A log snapshot has to describe the log data it carries: the log file's byte_size is where
// the file ends, and its log lines have to tile the file in order for their byte offsets to
// mean anything. pganalyze discards log lines that end past byte_size, so a byte_size that
// falls short of the lines means silently dropped log lines.
func TestLogSnapshotByteSizeMatchesLogLines(t *testing.T) {
	lineCount := 16
	server := state.MakeServer(config.ServerConfig{}, false)
	server.LogParser = logs.NewLogParser(logs.LogPrefixSimple, nil, false)
	logger := &util.Logger{Destination: log.New(os.Stderr, "", log.LstdFlags)}

	var content strings.Builder
	occurredAt := time.Date(2026, time.August, 3, 19, 37, 19, 0, time.UTC)
	for idx := range lineCount {
		// One backend per line, since log analysis groups the lines by backend PID
		fmt.Fprintf(&content, "%s [%d] ERROR:  duplicate key value violates unique constraint \"index_%d\"\n",
			occurredAt.Add(time.Duration(idx)*time.Second).Format("2006-01-02 15:04:05.000 MST"), 1000+idx, idx)
	}

	logLines, _ := logs.ParseAndAnalyzeBuffer(
		bufio.NewReader(strings.NewReader(content.String())), time.Time{}, server, state.CollectionOpts{}, logger,
	)
	if len(logLines) != lineCount {
		t.Fatalf("expected %d parsed log lines, got %d", lineCount, len(logLines))
	}

	logFile, err := state.NewLogFile("postgresql.log")
	if err != nil {
		t.Fatalf("could not initialize log file: %s", err)
	}
	logFile.LogLines = logLines
	logFile.UpdateByteSize()

	s, _ := transform.LogStateToLogSnapshot(server, state.TransientLogState{
		CollectedAt: occurredAt,
		LogFiles:    []state.LogFile{logFile},
	})

	byteSize := s.LogFileReferences[0].ByteSize
	if byteSize != int64(content.Len()) {
		t.Errorf("byte_size %d does not match the %d bytes of log lines in the snapshot", byteSize, content.Len())
	}

	var byteEnd int64
	for idx, logLine := range s.LogLineInformations {
		if logLine.ByteStart != byteEnd {
			t.Errorf("log line %d does not continue the previous one: byte_start %d, previous byte_end %d",
				idx, logLine.ByteStart, byteEnd)
		}
		byteEnd = logLine.ByteEnd

		if logLine.ByteEnd > byteSize {
			t.Errorf("log line %d ends past the file: byte_end %d, byte_size %d", idx, logLine.ByteEnd, byteSize)
		}
	}
}
