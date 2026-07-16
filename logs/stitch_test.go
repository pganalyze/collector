package logs

import (
	"io"
	"log"
	"strings"
	"testing"

	"github.com/pganalyze/collector/util"
)

func discardLogger() *util.Logger {
	return &util.Logger{Destination: log.New(io.Discard, "", 0)}
}

func TestStitchBufferAppends(t *testing.T) {
	s := NewStitchBuffer(discardLogger())
	s.Append("a")
	s.Append("b")
	s.Append("c")

	if got := s.String(); got != "abc" {
		t.Errorf("expected %q, got %q", "abc", got)
	}
}

// Content beyond MaxStitchedContentBytes is dropped rather than growing the buffer
// without bound, and Reset clears the accumulated content.
func TestStitchBufferCapsAndResets(t *testing.T) {
	orig := MaxStitchedContentBytes
	MaxStitchedContentBytes = 4096
	defer func() { MaxStitchedContentBytes = orig }()

	s := NewStitchBuffer(discardLogger())
	chunk := strings.Repeat("x", 1000)
	for i := 0; i < 5; i++ { // ~5 KB of content, just over the 4 KB cap
		s.Append(chunk)
	}

	if got := s.Len(); got > MaxStitchedContentBytes {
		t.Errorf("expected buffer capped at %d bytes, got %d", MaxStitchedContentBytes, got)
	}

	s.Reset()
	if got := s.Len(); got != 0 {
		t.Errorf("expected empty buffer after Reset, got %d bytes", got)
	}
}
