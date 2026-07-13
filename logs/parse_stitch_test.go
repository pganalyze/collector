package logs

import (
	"bufio"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const stitchTestPrefix = "2026-07-12 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  "

func parseStitchBuffer(input string) []state.LogLine {
	reader := bufio.NewReader(strings.NewReader(input))
	server := state.MakeServer(config.ServerConfig{}, false)
	server.LogParser = NewLogParser(LogPrefixAmazonRds, nil, false)
	logger := &util.Logger{Destination: log.New(io.Discard, "", 0)}
	logLines, _ := ParseAndAnalyzeBuffer(reader, time.Time{}, server, state.CollectionOpts{}, logger)
	return logLines
}

// A multi-line entry (e.g. an auto_explain plan) whose follow-on lines don't parse as
// new log lines must be stitched back onto its primary line's Content, in order.
func TestParseAndAnalyzeBufferStitchesMultiLine(t *testing.T) {
	multilineLog := `duration: 1.000 ms  plan:
			"Node": "a",
			"Node": "b",
			"Node": "c"
`

	logLines := parseStitchBuffer(stitchTestPrefix + multilineLog)

	if len(logLines) != 1 {
		t.Fatalf("expected 1 stitched log line, got %d", len(logLines))
	}
	if logLines[0].Content != multilineLog {
		t.Errorf("stitched content mismatch:\n got: %q\nmultilineLog: %q", logLines[0].Content, multilineLog)
	}
}

// A multi-line entry that's larger than the cap when stitched together, typically a large JSON EXPLAIN plan,
// must be bound by MaxStitchedContentBytes.
func TestParseAndAnalyzeBufferCapsRunawayStitch(t *testing.T) {
	orig := MaxStitchedContentBytes
	MaxStitchedContentBytes = 4096
	defer func() { MaxStitchedContentBytes = orig }()

	primary := "duration: 1.000 ms  plan:\n"
	additionalLine := strings.Repeat("x", 1000) + "\n"

	var b strings.Builder
	b.WriteString(stitchTestPrefix + primary)
	for i := 0; i < 5; i++ { // ~5 KB of continuation, just over the 4 KB cap
		b.WriteString(additionalLine)
	}

	logLines := parseStitchBuffer(b.String())

	if len(logLines) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(logLines))
	}
	// Content is the primary line plus the additional lines
	maxContentBytes := len(primary) + MaxStitchedContentBytes
	if totalLogLineBytes := len(logLines[0].Content); totalLogLineBytes > maxContentBytes {
		t.Errorf("expected stitched content capped at %d bytes, got %d", maxContentBytes, totalLogLineBytes)
	}
}
