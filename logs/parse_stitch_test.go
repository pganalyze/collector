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

const stitchTestPrefix = "2018-03-11 20:00:02 UTC:1.1.1.1(2):a@b:[3]:LOG:  "

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
	input := stitchTestPrefix + "duration: 1.000 ms  plan:\n" +
		"  \"Node\": \"a\",\n" +
		"  \"Node\": \"b\",\n" +
		"  \"Node\": \"c\"\n"

	logLines := parseStitchBuffer(input)

	if len(logLines) != 1 {
		t.Fatalf("expected 1 stitched log line, got %d", len(logLines))
	}
	want := "duration: 1.000 ms  plan:\n  \"Node\": \"a\",\n  \"Node\": \"b\",\n  \"Node\": \"c\"\n"
	if logLines[0].Content != want {
		t.Errorf("stitched content mismatch:\n got: %q\nwant: %q", logLines[0].Content, want)
	}
}

// A pathologically large multi-line entry (a runaway plan, or a log_line_prefix
// mismatch that makes every line look like a continuation) must not grow the stitched
// Content without bound; it is capped at maxAdditionalLinesBytes.
func TestParseAndAnalyzeBufferCapsRunawayStitch(t *testing.T) {
	orig := maxAdditionalLinesBytes
	maxAdditionalLinesBytes = 4096
	defer func() { maxAdditionalLinesBytes = orig }()

	var b strings.Builder
	b.WriteString(stitchTestPrefix + "duration: 1.000 ms  plan:\n")
	contLine := strings.Repeat("x", 1000) + "\n"
	for i := 0; i < 200; i++ { // ~200 KB of continuation, far above the 4 KB cap
		b.WriteString(contLine)
	}

	logLines := parseStitchBuffer(b.String())

	if len(logLines) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(logLines))
	}
	// Content should be bounded near the cap (primary line + up to the cap of
	// continuation), not the full ~200 KB of input.
	if got := len(logLines[0].Content); got > maxAdditionalLinesBytes+len(contLine) {
		t.Errorf("expected stitched content capped near %d bytes, got %d", maxAdditionalLinesBytes, got)
	}
}
