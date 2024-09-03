package logs

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

type PrefixEscape struct {
	Regexp     string
	ApplyValue func(value string, logLine *state.LogLine, tz *time.Location)
	// Indicates a value may not always be present for this escape (e.g., when logging from a non-backend process)
	Optional bool
}

// This is a map of the various log_line_prefix format strings; see
// https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-LINE-PREFIX
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
			logLine.OccurredAt = GetOccurredAt(value, tz, false)
		},
	},
	// Time stamp with milliseconds
	'm': {
		Regexp: `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3} (?:[A-Z]{1,4}|[+-]\d+)`,
		ApplyValue: func(value string, logLine *state.LogLine, tz *time.Location) {
			logLine.OccurredAt = GetOccurredAt(value, tz, false)
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

	lineRegexpWithoutLogLevel *regexp.Regexp
}

func NewLogParser(prefix string, tz *time.Location, isSyslog bool) *LogParser {
	prefixRegexp, prefixElements := parsePrefix(prefix)
	lineRegexp := regexp.MustCompile("(?ms)^" + prefixRegexp + `(\w+):\s+(.*\n?)$`)
	lineRegexpWithoutLogLevel := regexp.MustCompile("(?ms)^" + prefixRegexp + `(.*\n?)$`)
	return &LogParser{
		prefix:   prefix,
		tz:       tz,
		isSyslog: isSyslog,

		lineRegexp:     lineRegexp,
		prefixElements: prefixElements,

		lineRegexpWithoutLogLevel: lineRegexpWithoutLogLevel,
	}
}

func getLogConfigFromSettings(settings []state.PostgresSetting) (tz *time.Location, prefix string) {
	for _, setting := range settings {
		if !setting.ResetValue.Valid {
			continue
		}

		if setting.Name == "log_timezone" {
			zoneStr := setting.ResetValue.String
			zone, err := time.LoadLocation(zoneStr)
			if err == nil {
				tz = zone
			}
		} else if setting.Name == "log_line_prefix" {
			prefix = setting.ResetValue.String
		}
	}
	return
}

func SyncLogParser(server *state.Server, settings []state.PostgresSetting) {
	server.LogParseMutex.RLock()

	tz, prefix := getLogConfigFromSettings(settings)
	isSyslog := server.Config.LogSyslogServer != ""
	parserInSync := server.LogParser != nil && server.LogParser.Matches(prefix, tz, isSyslog)
	server.LogParseMutex.RUnlock()

	if parserInSync {
		return
	}

	server.LogParseMutex.Lock()
	defer server.LogParseMutex.Unlock()

	server.LogParser = NewLogParser(prefix, tz, isSyslog)
}

func (lp *LogParser) ValidatePrefix() error {
	dbInPrefix, err := regexp.MatchString("(?:^|[^%])%d", lp.prefix)
	if err != nil {
		return fmt.Errorf("could not check: %s", err)
	}
	userInPrefix, err := regexp.MatchString("(?:^|[^%])%u", lp.prefix)
	if err != nil {
		return fmt.Errorf("could not check: %s", err)
	}
	if !dbInPrefix && !userInPrefix {
		return errors.New("database (%d) and user (%u) not found: pganalyze will not be able to correctly classify some log lines")
	} else if !dbInPrefix {
		return errors.New("database (%d) not found: pganalyze will not be able to correctly classify some log lines")
	} else if !userInPrefix {
		return errors.New("user (%u) not found: pganalyze will not be able to correctly classify some log lines")
	} else {
		return nil
	}
}

func (lp *LogParser) Matches(prefix string, tz *time.Location, isSyslog bool) bool {
	return lp.prefix == prefix && tz.String() == lp.tz.String() && lp.isSyslog == isSyslog
}

func (lp *LogParser) GetOccurredAt(timePart string) time.Time {
	if lp.tz != nil && !lp.isSyslog {
		lastSpaceIdx := strings.LastIndex(timePart, " ")
		if lastSpaceIdx == -1 {
			return time.Time{}
		}
		timePartNoTz := timePart[0:lastSpaceIdx]
		result, err := time.ParseInLocation("2006-01-02 15:04:05", timePartNoTz, lp.tz)
		if err != nil {
			return time.Time{}
		}

		return result
	}

	// Assume Postgres time format unless overriden by the prefix (e.g. syslog)
	var timeFormat, timeFormatAlt string
	if lp.isSyslog {
		timeFormat = "2006 Jan  2 15:04:05"
		timeFormatAlt = ""
	} else {
		timeFormat = "2006-01-02 15:04:05 -0700"
		timeFormatAlt = "2006-01-02 15:04:05 MST"
	}

	ts, err := time.Parse(timeFormat, timePart)
	if err != nil {
		if timeFormatAlt != "" {
			// Ensure we have the correct format remembered for ParseInLocation call that may happen later
			timeFormat = timeFormatAlt
			ts, err = time.Parse(timeFormat, timePart)
		}
		if err != nil {
			return time.Time{}
		}
	}

	// Handle non-UTC timezones in systems that have log_timezone set to a different
	// timezone value than their system timezone. This is necessary because Go otherwise
	// only reads the timezone name but does not set the timezone offset, see
	// https://pkg.go.dev/time#Parse
	zone, offset := ts.Zone()
	if offset == 0 && zone != "UTC" && zone != "" {
		var zoneLocation *time.Location
		zoneNum, err := strconv.Atoi(zone)
		if err == nil {
			zoneLocation = time.FixedZone(zone, zoneNum*3600)
		} else {
			zoneLocation, err = time.LoadLocation(zone)
			if err != nil {
				// We don't know which timezone this is (and a timezone name is present), so we can't process this log line
				return time.Time{}
			}
		}
		ts, err = time.ParseInLocation(timeFormat, timePart, zoneLocation)
		if err != nil {
			// Technically this should not occur (as we should have already failed previously in time.Parse)
			return time.Time{}
		}
	}
	return ts
}

func parseSyslogLine(line string, tz *time.Location) (logLine state.LogLine, ok bool) {
	parts := RsyslogRegexp.FindStringSubmatch(line)
	if len(parts) == 0 {
		return
	}

	timePart := fmt.Sprintf("%d %s", time.Now().Year(), parts[1])
	// ignore syslog hostname
	// ignore syslog process name
	pidPart := parts[4]
	// ignore syslog postgres sequence and split number
	levelPart := parts[6]
	contentPart := strings.Replace(parts[7], "#011", "\t", -1)

	parts = LogPrefixNoTimestampUserDatabaseAppRegexp.FindStringSubmatch(contentPart)
	if len(parts) == 6 {
		userPart := parts[1]
		dbPart := parts[2]
		appPart := parts[3]
		levelPart = parts[4]
		contentPart = parts[5]

		if userPart != "[unknown]" {
			logLine.Username = userPart
		}
		if dbPart != "[unknown]" {
			logLine.Database = dbPart
		}
		if appPart != "[unknown]" {
			logLine.Application = appPart
		}
	}

	occurredAt := GetOccurredAt(timePart, tz, true)
	if occurredAt.IsZero() {
		return
	}

	logLine.OccurredAt = occurredAt

	backendPid, _ := strconv.ParseInt(pidPart, 10, 32)
	logLine.BackendPid = int32(backendPid)
	logLine.Content = contentPart

	// This is actually a continuation of a previous line
	if levelPart == "" {
		return
	}

	logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[levelPart])
	ok = true

	return
}

func (lp *LogParser) ParseLine(line string) (logLine state.LogLine, ok bool) {
	if lp.isSyslog {
		return parseSyslogLine(line, lp.tz)
	}

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

func (lp *LogParser) GetPrefixAndContent(line string) (prefix string, content string, ok bool) {
	// Last 2 indexes here is important
	// [..., end idx of prefix, start idx of content, end idx of content]
	matchIdxs := lp.lineRegexpWithoutLogLevel.FindStringSubmatchIndex(line)
	if matchIdxs == nil {
		return "", "", false
	}
	contentStart := matchIdxs[len(matchIdxs)-2]
	contentEnd := matchIdxs[len(matchIdxs)-1]
	return line[0:contentStart], line[contentStart:contentEnd], true
}
