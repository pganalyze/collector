package stream

import (
	"io"
	"log"
	"testing"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func discardLogger() *util.Logger {
	return &util.Logger{Destination: log.New(io.Discard, "", 0)}
}

// Continuation lines (LogLevel UNKNOWN) must be stitched back onto the preceding
// analyzable line's Content, in order.
func TestStitchLogLinesConcatenatesContinuations(t *testing.T) {
	readyLogLines := []state.LogLine{
		{LogLevel: pganalyze_collector.LogLineInformation_LOG, Content: `duration: 1.000 ms  plan:
`},
		{LogLevel: pganalyze_collector.LogLineInformation_UNKNOWN, Content: `  "Node": "a",
`},
		{LogLevel: pganalyze_collector.LogLineInformation_UNKNOWN, Content: `  "Node": "b"
`},
	}

	out := stitchLogLines(readyLogLines, discardLogger())

	if len(out) != 1 {
		t.Fatalf("expected 1 stitched log line, got %d", len(out))
	}
	want := `duration: 1.000 ms  plan:
  "Node": "a",
  "Node": "b"
`
	if out[0].Content != want {
		t.Errorf("stitched content mismatch:\n got: %q\nwant: %q", out[0].Content, want)
	}
}
