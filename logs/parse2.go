package logs

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func IsSupportedPrefix2(prefix string) bool {
	// This is not true—some prefixes could produce metadata too ambiguous for us
	// to parse—but close enough
	return true
}

type PrefixEscape struct {
	Regexp     string
	ApplyValue func(value string, logLine *state.LogLine, tz *time.Location)
	// Indicates a value may not always be present for this escape (e.g., when logging from a non-backend process)
	Optional bool
}

// not included: %q and %%, which are easier to handle by special-casing
var EscapeMatchers = map[rune]PrefixEscape{
	// Application name
	'a': {
		Regexp: `.+?`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			if value == "[unknown]" {
				return
			}
			logLine.Application = value
		},
		Optional: true,
	},
	// User name
	'u': {
		Regexp: `.+?`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			if value == "[unknown]" {
				return
			}
			logLine.Username = value
		},
		Optional: true,
	},
	// Database name
	'd': {
		Regexp: `.+?`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			if value == "[unknown]" {
				return
			}
			logLine.Database = value
		},
		Optional: true,
	},
	// Remote host name or IP address, and remote port
	'r': {
		Regexp:   `[a-zA-Z0-9:.-]+\(\d{1,5}\)|\[local\]`,
		Optional: true,
	},
	// Remote host name or IP address
	'h': {
		Regexp:   `[a-zA-Z0-9:.-]+|\[local\]`,
		Optional: true,
	},
	// Backend type
	'b': {
		Regexp: `[a-z ]+`,
	},
	// 	Process ID
	'p': {
		Regexp: `\d+`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			intVal, _ := strconv.ParseInt(value, 10, 32)
			logLine.BackendPid = int32(intVal)
		},
	},
	// 	Process ID of the parallel group leader, if this process is a parallel query worker
	'P': {
		Regexp:   `\d+`,
		Optional: true,
	},
	// Time stamp without milliseconds
	't': {
		Regexp: `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} (?:[A-Z]{1,4}|[+-]\d+)`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			logLine.OccurredAt = getOccurredAt(value, tz, false)
		},
	},
	// Time stamp with milliseconds
	'm': {
		Regexp: `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3} (?:[A-Z]{1,4}|[+-]\d+)`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			logLine.OccurredAt = getOccurredAt(value, tz, false)
		},
	},
	// Time stamp with milliseconds (as a Unix epoch)
	'n': {
		Regexp: `\d+\.\d+`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			tsparts := strings.SplitN(value, ".", 2)
			seconds, _ := strconv.ParseInt(tsparts[0], 10, 64)
			millis, _ := strconv.ParseInt(tsparts[1], 10, 64)
			logLine.OccurredAt = time.Unix(seconds, millis*1_000_000)
		},
	},
	// Command tag: type of session's current command
	'i': {
		Regexp:   `[A-Z_ ]+`,
		Optional: true,
	},
	// SQLSTATE error code
	'e': {
		Regexp: `[0-9A-Z]{5}`,
	},
	// Session ID: see below
	'c': {
		Regexp:   `[0-9a-f]{1,8}\.[0-9a-f]{1,8}`,
		Optional: true,
	},
	// Number of the log line for each session or process, starting at 1
	'l': {
		Regexp: `\d+`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			intVal, _ := strconv.ParseInt(value, 10, 32)
			logLine.LogLineNumber = int32(intVal)
		},
	},
	// Process start time stamp
	's': {
		Regexp: `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} (?:[A-Z]{1,4}|[+-]\d+)`,
	},
	// Virtual transaction ID (backendID/localXID); see Section 74.1
	'v': {
		Regexp:   `\d+\/\d+`,
		Optional: true,
	},
	// Transaction ID (0 if none is assigned); see Section 74.1
	'x': {
		Regexp:   `\d+`,
		Optional: true,
	},
	// Query identifier of the current query. Query identifiers are not computed by default, so this field will be zero unless compute_query_id parameter is enabled or a third-party module that computes query identifiers is configured.
	'Q': {
		Regexp: `-?\d+`,
	},
}

type LogParser struct {
	prefix   string
	tz       *time.Location
	isSyslog bool

	lineRegexp     *regexp.Regexp
	prefixElements []PrefixEscape
}

func NewLogParser(prefix string, tz *time.Location, isSyslog bool) *LogParser {
	prefixRegexp, prefixElements := parsePrefix(prefix)
	lineRegexp := regexp.MustCompile("(?ms)^" + prefixRegexp + `(\w+):\s+(.*\n?)$`)
	return &LogParser{
		prefix:   prefix,
		tz:       tz,
		isSyslog: isSyslog,

		lineRegexp:     lineRegexp,
		prefixElements: prefixElements,
	}
}

func (lp *LogParser) ParseLine(line string) (logLine state.LogLine, ok bool) {
	if lp.prefix == "" {
		return logLine, false
	}

	lineValues := lp.lineRegexp.FindStringSubmatch(line)

	if lineValues == nil {
		// If this is an unprefixed line, it may be a continuation of a previous line
		logLine.Content = line
		return logLine, false
	}

	for i, elem := range lp.prefixElements {
		if elem.ApplyValue != nil {
			value := lineValues[i+1]
			elem.ApplyValue(value, &logLine, lp.tz)
		}
	}

	levelPart := lineValues[len(lineValues)-2]
	logLine.Content = lineValues[len(lineValues)-1]
	logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[levelPart])

	return logLine, true
}

func parsePrefix(prefix string) (string, []PrefixEscape) {
	var escapes []PrefixEscape
	var resultRegexp strings.Builder
	// for when %q is used
	var pastq = false

	prefixLen := len(prefix)
	var runeValue rune
	for byteIdx, width := 0, 0; byteIdx < prefixLen; byteIdx += width {
		runeValue, width = utf8.DecodeRuneInString(prefix[byteIdx:])
		if runeValue != '%' || byteIdx == prefixLen-1 {
			// keep in regexp to match as a literal, but ignore
			resultRegexp.WriteString(regexp.QuoteMeta(string(runeValue)))
			continue
		}

		// at this point we have an escape to handle; check the actual escape code
		// value first
		byteIdx += width
		runeValue, width = utf8.DecodeRuneInString(prefix[byteIdx:])
		if runeValue == '%' {
			// if we see another %, it's escaped so we should expect it in the string
			resultRegexp.WriteRune('%')
			continue
		}

		// flag %q if necessary: we wrap the rest of the expression until the end of
		// the log_line_prefix in an optional non-capturing group
		if !pastq && runeValue == 'q' {
			pastq = true
			resultRegexp.WriteString("(?:")
			continue
		}

		escape, ok := EscapeMatchers[runeValue]
		if !ok {
			// escapes that don't correspond to known escape codes are ignored
			continue
		}

		escapes = append(escapes, escape)
		resultRegexp.WriteString("(")
		resultRegexp.WriteString(escape.Regexp)
		resultRegexp.WriteString(")")
		if escape.Optional {
			resultRegexp.WriteString("?")
		}
		// TODO: some groups may be empty for some backend types; add a '?' to the
		// regexp for those cases?
	}

	if pastq {
		resultRegexp.WriteString(")?")
	}

	return resultRegexp.String(), escapes
}

func GetHerokuLogLinePrefix(logLine string) string {
	if strings.Contains(logLine, "time_ms = ") {
		return ` sql_error_code = %e time_ms = "%m" pid="%p" proc_start_time="%s" session_id="%c" vtid="%v" tid="%x" log_line="%l" %qdatabase="%d" connection_source="%r" user="%u" application_name="%a" `
	} else {
		// older prefix; we may be able to drop this at some point
		return " sql_error_code = %e "
	}
}
