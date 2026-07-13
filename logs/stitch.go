package logs

import (
	"strings"

	"github.com/pganalyze/collector/util"
)

// MaxStitchedContentBytes bounds how much continuation content is stitched onto a single
// log line. Multi-line entries (e.g. auto_explain JSON plans) are stitched onto their
// primary line, and a log_line_prefix mismatch can likewise make every follow-on line
// look like a continuation. Without a bound, a single Content string can grow without
// limit and OOM the collector. It is a var so tests can lower it.
var MaxStitchedContentBytes = 10 * 1024 * 1024

// StitchBuffer accumulates continuation-line content destined for a single parent log
// line, capped at MaxStitchedContentBytes. Content beyond the cap is dropped, with a
// single warning emitted per buffer lifetime (until Reset).
type StitchBuffer struct {
	builder   strings.Builder
	truncated bool
	logger    *util.Logger
}

func NewStitchBuffer(logger *util.Logger) *StitchBuffer {
	return &StitchBuffer{logger: logger}
}

// Grow hints the expected total content size, bounded by the cap so a pathologically
// long run of continuation lines can't trigger a large up-front allocation.
func (s *StitchBuffer) Grow(n int) {
	s.builder.Grow(min(n, MaxStitchedContentBytes))
}

// Append adds continuation content, dropping it (and warning once) if it would grow the
// buffer past MaxStitchedContentBytes.
func (s *StitchBuffer) Append(content string) {
	if s.builder.Len()+len(content) > MaxStitchedContentBytes {
		if !s.truncated {
			s.truncated = true
			s.logger.PrintWarning("Log line continuation exceeded %d bytes and was truncated; "+
				"a single log entry (e.g. an auto_explain JSON plan) or a log_line_prefix "+
				"mismatch is producing more continuation content than the collector will "+
				"stitch together", MaxStitchedContentBytes)
		}
		return
	}
	s.builder.WriteString(content)
}

func (s *StitchBuffer) Len() int {
	return s.builder.Len()
}

func (s *StitchBuffer) String() string {
	return s.builder.String()
}

func (s *StitchBuffer) Reset() {
	s.builder.Reset()
	s.truncated = false
}
