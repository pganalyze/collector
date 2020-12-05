package logs

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	uuid "github.com/satori/go.uuid"
)

const LogPrefixAmazonRds string = "%t:%r:%u@%d:[%p]:"
const LogPrefixAzure string = "%t-%c-"
const LogPrefixCustom1 string = "%m [%p][%v] : [%l-1] %q[app=%a] "
const LogPrefixCustom2 string = "%t [%p-%l] %q%u@%d "
const LogPrefixCustom3 string = "%m [%p] %q[user=%u,db=%d,app=%a] "
const LogPrefixCustom4 string = "%m [%p] %q[user=%u,db=%d,app=%a,host=%h] "
const LogPrefixCustom5 string = "%t [%p]: [%l-1] user=%u,db=%d - PG-%e "
const LogPrefixCustom6 string = "%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h "
const LogPrefixCustom7 string = "%t [%p]: [%l-1] [trx_id=%x] user=%u,db=%d "
const LogPrefixCustom8 string = "[%p]: [%l-1] db=%d,user=%u "
const LogPrefixCustom9 string = "%m %r %u %a [%c] [%p] "
const LogPrefixCustom10 string = "%m [%p]: [%l-1] db=%d,user=%u "
const LogPrefixSimple string = "%m [%p] "
const LogPrefixEmpty string = ""

var SupportedPrefixes = []string{
	LogPrefixAmazonRds, LogPrefixAzure, LogPrefixCustom1, LogPrefixCustom2,
	LogPrefixCustom3, LogPrefixCustom4, LogPrefixCustom5, LogPrefixCustom6,
	LogPrefixCustom7, LogPrefixCustom8, LogPrefixCustom9, LogPrefixCustom10,
	LogPrefixSimple, LogPrefixEmpty,
}

// Every one of these regexps should produce exactly one matching group
var TimeRegexp = `(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)? [\-+]?\w+)` // %t or %m
var HostAndPortRegexp = `(.+(?:\(\d+\))?)?`                                  // %r
var PidRegexp = `(\d+)`                                                      // %p
var UserRegexp = `(\S*)`                                                     // %u
var DbRegexp = `(\S*)`                                                       // %d
var AppRegexp = `(\S*)`                                                      // %a
var AppInsideBracketsRegexp = `(\[unknown\]|[^,\]]*)`                        // %a
var HostRegexp = `(\S*)`                                                     // %h
var VirtualTxRegexp = `(\d+/\d+)?`                                           // %v
var LogLineCounterRegexp = `(\d+)`                                           // %l
var SqlstateRegexp = `(\w{5})`                                               // %e
var TransactionIdRegexp = `(\d+)`                                            // %x
var SessionIdRegexp = `(\w+\.\w+)`                                           // %c
// Missing:
// - %n (unix timestamp)
// - %i (command tag)
// - %s (process start timestamp)

var LevelAndContentRegexp = `(\w+):\s+(.*\n?)$`
var LogPrefixAmazonRdsRegexp = regexp.MustCompile(`(?s)^` + TimeRegexp + `:` + HostAndPortRegexp + `:` + UserRegexp + `@` + DbRegexp + `:\[` + PidRegexp + `\]:` + LevelAndContentRegexp)
var LogPrefixAzureRegexp = regexp.MustCompile(`(?s)^` + TimeRegexp + `-` + SessionIdRegexp + `-` + LevelAndContentRegexp)
var LogPrefixCustom1Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]\[` + VirtualTxRegexp + `\] : \[` + LogLineCounterRegexp + `-1\] (?:\[app=` + AppInsideBracketsRegexp + `\] )?` + LevelAndContentRegexp)
var LogPrefixCustom2Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `-` + LogLineCounterRegexp + `\] ` + `(?:` + UserRegexp + `@` + DbRegexp + ` )?` + LevelAndContentRegexp)
var LogPrefixCustom3Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\] (?:\[user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppInsideBracketsRegexp + `\] )?` + LevelAndContentRegexp)
var LogPrefixCustom4Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\] (?:\[user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppInsideBracketsRegexp + `,host=` + HostRegexp + `\] )?` + LevelAndContentRegexp)
var LogPrefixCustom5Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] user=` + UserRegexp + `,db=` + DbRegexp + ` - PG-` + SqlstateRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom6Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppInsideBracketsRegexp + `,client=` + HostRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom7Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] \[trx_id=` + TransactionIdRegexp + `\] user=` + UserRegexp + `,db=` + DbRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom8Regexp = regexp.MustCompile(`(?s)^\[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] db=` + DbRegexp + `,user=` + UserRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom9Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` ` + HostAndPortRegexp + ` ` + UserRegexp + ` ` + AppRegexp + ` \[` + SessionIdRegexp + `\] \[` + PidRegexp + `\] ` + LevelAndContentRegexp)
var LogPrefixCustom10Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] db=` + DbRegexp + `,user=` + UserRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixSimpleRegexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\] ` + LevelAndContentRegexp)
var LogPrefixNoTimestampUserDatabaseAppRegexp = regexp.MustCompile(`(?s)^\[user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppInsideBracketsRegexp + `\] ` + LevelAndContentRegexp)

var SyslogSequenceAndSplitRegexp = `(\[[\d-]+\])?`

var RsyslogLevelAndContentRegexp = `(?:(\w+):\s+)?(.*\n?)$`
var RsyslogTimeRegexp = `(\w+\s+\d+ \d{2}:\d{2}:\d{2})`
var RsyslogHostnameRegxp = `(\S+)`
var RsyslogProcessNameRegexp = `(\w+)`
var RsyslogRegexp = regexp.MustCompile(`^` + RsyslogTimeRegexp + ` ` + RsyslogHostnameRegxp + ` ` + RsyslogProcessNameRegexp + `\[` + PidRegexp + `\]: ` + SyslogSequenceAndSplitRegexp + ` ` + RsyslogLevelAndContentRegexp)

// The Heroku log_line_prefix is handled directly in the Heroku log receiver, included here for reference only
var HerokuLogLinePrefix = " sql_error_code = %e "
var HerokuPostgresDebugRegexp = regexp.MustCompile(`^(\w+ \d+ \d+:\d+:\d+ \w+ app\[postgres\] \w+ )?\[(\w+)\] \[\d+-\d+\] ( sql_error_code = ` + SqlstateRegexp + ` (\w+):  )?(.+)`)

func IsSupportedPrefix(prefix string) bool {
	for _, supportedPrefix := range SupportedPrefixes {
		if supportedPrefix == prefix {
			return true
		}
	}
	return false
}

func ParseLogLineWithPrefix(prefix string, line string) (logLine state.LogLine, ok bool) {
	var timePart, userPart, dbPart, appPart, pidPart, logLineNumberPart, levelPart, contentPart string

	// Assume Postgres time format unless overriden by the prefix (e.g. syslog)
	timeFormat := "2006-01-02 15:04:05 -0700"
	timeFormatAlt := "2006-01-02 15:04:05 MST"

	rsyslog := false

	if prefix == "" {
		if LogPrefixAmazonRdsRegexp.MatchString(line) {
			prefix = LogPrefixAmazonRds
		} else if LogPrefixAzureRegexp.MatchString(line) {
			prefix = LogPrefixAzure
		} else if LogPrefixCustom1Regexp.MatchString(line) {
			prefix = LogPrefixCustom1
		} else if LogPrefixCustom2Regexp.MatchString(line) {
			prefix = LogPrefixCustom2
		} else if LogPrefixCustom4Regexp.MatchString(line) { // 4 is more specific than 3, so needs to go first
			prefix = LogPrefixCustom4
		} else if LogPrefixCustom3Regexp.MatchString(line) {
			prefix = LogPrefixCustom3
		} else if LogPrefixCustom5Regexp.MatchString(line) {
			prefix = LogPrefixCustom5
		} else if LogPrefixCustom6Regexp.MatchString(line) {
			prefix = LogPrefixCustom6
		} else if LogPrefixCustom7Regexp.MatchString(line) {
			prefix = LogPrefixCustom7
		} else if LogPrefixCustom8Regexp.MatchString(line) {
			prefix = LogPrefixCustom8
		} else if LogPrefixCustom9Regexp.MatchString(line) {
			prefix = LogPrefixCustom9
		} else if LogPrefixCustom10Regexp.MatchString(line) {
			prefix = LogPrefixCustom10
		} else if LogPrefixSimpleRegexp.MatchString(line) {
			prefix = LogPrefixSimple
		} else if RsyslogRegexp.MatchString(line) {
			rsyslog = true
		}
	}

	if rsyslog {
		parts := RsyslogRegexp.FindStringSubmatch(line)
		if len(parts) == 0 {
			return
		}
		timeFormat = "2006 Jan  2 15:04:05"
		timeFormatAlt = ""
		timePart = fmt.Sprintf("%d %s", time.Now().Year(), parts[1])
		// ignore syslog hostname
		// ignore syslog process name
		pidPart = parts[4]
		// ignore syslog postgres sequence and split number
		levelPart = parts[6]
		contentPart = strings.Replace(parts[7], "#011", "\t", -1)

		parts = LogPrefixNoTimestampUserDatabaseAppRegexp.FindStringSubmatch(contentPart)
		if len(parts) == 6 {
			userPart = parts[1]
			dbPart = parts[2]
			appPart = parts[3]
			levelPart = parts[4]
			contentPart = parts[5]
		}
	} else {
		switch prefix {
		case LogPrefixAmazonRds: // "%t:%r:%u@%d:[%p]:"
			parts := LogPrefixAmazonRdsRegexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}

			timePart = parts[1]
			// skip %r (ip+port)
			userPart = parts[3]
			dbPart = parts[4]
			pidPart = parts[5]
			levelPart = parts[6]
			contentPart = parts[7]
		case LogPrefixAzure: // "%t-%c-"
			parts := LogPrefixAzureRegexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}

			timePart = parts[1]
			// skip %c (session id)
			levelPart = parts[3]
			contentPart = parts[4]
		case LogPrefixCustom1: // "%m [%p][%v] : [%l-1] %q[app=%a] "
			parts := LogPrefixCustom1Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			// skip %v (virtual TX)
			logLineNumberPart = parts[4]
			appPart = parts[5]
			levelPart = parts[6]
			contentPart = parts[7]
		case LogPrefixCustom2: // "%t [%p-1] %q%u@%d "
			parts := LogPrefixCustom2Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			logLineNumberPart = parts[3]
			userPart = parts[4]
			dbPart = parts[5]
			levelPart = parts[6]
			contentPart = parts[7]
		case LogPrefixCustom3: // "%m [%p] %q[user=%u,db=%d,app=%a] ""
			parts := LogPrefixCustom3Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			userPart = parts[3]
			dbPart = parts[4]
			appPart = parts[5]
			levelPart = parts[6]
			contentPart = parts[7]
		case LogPrefixCustom4: // "%m [%p] %q[user=%u,db=%d,app=%a,host=%h] "
			parts := LogPrefixCustom4Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			userPart = parts[3]
			dbPart = parts[4]
			appPart = parts[5]
			// skip %h (host)
			levelPart = parts[7]
			contentPart = parts[8]
		case LogPrefixCustom5: // "%t [%p]: [%l-1] user=%u,db=%d - PG-%e "
			parts := LogPrefixCustom5Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			logLineNumberPart = parts[3]
			userPart = parts[4]
			dbPart = parts[5]
			// skip %e (SQLSTATE)
			levelPart = parts[7]
			contentPart = parts[8]
		case LogPrefixCustom6: // "%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h "
			parts := LogPrefixCustom6Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			logLineNumberPart = parts[3]
			userPart = parts[4]
			dbPart = parts[5]
			// skip %a (application name)
			// skip %h (host)
			levelPart = parts[8]
			contentPart = parts[9]
		case LogPrefixCustom7: // "%t [%p]: [%l-1] [trx_id=%x] user=%u,db=%d "
			parts := LogPrefixCustom7Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			logLineNumberPart = parts[3]
			// skip %x (transaction id)
			userPart = parts[5]
			dbPart = parts[6]
			levelPart = parts[7]
			contentPart = parts[8]
		case LogPrefixCustom8: // "[%p]: [%l-1] db=%d,user=%u "
			parts := LogPrefixCustom8Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			pidPart = parts[1]
			logLineNumberPart = parts[2]
			dbPart = parts[3]
			userPart = parts[4]
			levelPart = parts[5]
			contentPart = parts[6]
		case LogPrefixCustom9: // "%m %r %u %a [%c] [%p] "
			parts := LogPrefixCustom9Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			// skip %r (ip+port)
			userPart = parts[3]
			appPart = parts[4]
			// skip %c (session id)
			pidPart = parts[6]
			levelPart = parts[7]
			contentPart = parts[8]
		case LogPrefixCustom10: // "%t [%p]: [%l-1] db=%d,user=%u "
			parts := LogPrefixCustom10Regexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			logLineNumberPart = parts[3]
			dbPart = parts[4]
			userPart = parts[5]
			levelPart = parts[6]
			contentPart = parts[7]
		case LogPrefixSimple: // "%t [%p] "
			parts := LogPrefixSimpleRegexp.FindStringSubmatch(line)
			if len(parts) == 0 {
				return
			}
			timePart = parts[1]
			pidPart = parts[2]
			levelPart = parts[3]
			contentPart = parts[4]
		default:
			// Some callers use the content of unparsed lines to stitch multi-line logs together
			logLine.Content = line
			return
		}
	}

	var err error
	if timePart != "" {
		logLine.OccurredAt, err = time.Parse(timeFormat, timePart)
		if err != nil {
			if timeFormatAlt != "" {
				logLine.OccurredAt, err = time.Parse(timeFormatAlt, timePart)
			}
			if err != nil {
				ok = false
				return
			}
		}
	}

	if userPart != "[unknown]" {
		logLine.Username = userPart
	}
	if dbPart != "[unknown]" {
		logLine.Database = dbPart
	}
	if appPart != "[unknown]" {
		logLine.Application = appPart
	}
	if logLineNumberPart != "" {
		logLineNumber, _ := strconv.Atoi(logLineNumberPart)
		logLine.LogLineNumber = int32(logLineNumber)
	}

	backendPid, _ := strconv.Atoi(pidPart)
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

func ParseAndAnalyzeBuffer(buffer string, initialByteStart int64, linesNewerThan time.Time) ([]state.LogLine, []state.PostgresQuerySample, int64) {
	var logLines []state.LogLine
	currentByteStart := initialByteStart
	reader := bufio.NewReader(strings.NewReader(buffer))

	for {
		line, err := reader.ReadString('\n')
		byteStart := currentByteStart
		currentByteStart += int64(len(line))

		// This is intentionally after updating currentByteStart, since we consume the
		// data in the file even if an error is returned
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Log Read ERROR: %s", err)
			}
			break
		}

		logLine, ok := ParseLogLineWithPrefix("", line)
		if !ok {
			// Assume that a parsing error in a follow-on line means that we actually
			// got additional data for the previous line
			if len(logLines) > 0 && logLine.Content != "" {
				logLines[len(logLines)-1].Content += logLine.Content
				logLines[len(logLines)-1].ByteEnd += int64(len(logLine.Content))
			}
			continue
		}

		// Ignore loglines which are outside our time window
		if logLine.OccurredAt.Before(linesNewerThan) {
			continue
		}

		logLine.ByteStart = byteStart
		logLine.ByteContentStart = byteStart + int64(len(line)-len(logLine.Content))
		logLine.ByteEnd = byteStart + int64(len(line))

		// Generate unique ID that can be used to reference this line
		logLine.UUID = uuid.NewV4()

		logLines = append(logLines, logLine)
	}

	newLogLines, newSamples := AnalyzeLogLines(logLines)
	return newLogLines, newSamples, currentByteStart
}

func DebugParseAndAnalyzeBuffer(buffer string) ([]state.LogLine, []state.PostgresQuerySample) {
	var logLines []state.LogLine
	currentByteStart := int64(0)
	reader := bufio.NewReader(strings.NewReader(buffer))

	for {
		line, err := reader.ReadString('\n')
		byteStart := currentByteStart
		currentByteStart += int64(len(line))

		// This is intentionally after updating currentByteStart, since we consume the
		// data in the file even if an error is returned
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Log Read ERROR: %s", err)
			}
			break
		}

		contentParts := HerokuPostgresDebugRegexp.FindStringSubmatch(line)
		var logLine state.LogLine
		if len(contentParts) == 7 {
			logLine.Content = contentParts[6]
			if contentParts[4] != "" && contentParts[5] != "" { // We have a SQLSTATE and a log level, so its a new Postgres log line
				logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[contentParts[5]])
			} else {
				logLines[len(logLines)-1].Content += logLine.Content
				logLines[len(logLines)-1].ByteEnd += int64(len(logLine.Content))
				continue
			}
		} else {
			var ok bool
			logLine, ok = ParseLogLineWithPrefix("", line)
			if !ok {
				// Assume that a parsing error in a follow-on line means that we actually
				// got additional data for the previous line
				if len(logLines) > 0 && logLine.Content != "" {
					logLines[len(logLines)-1].Content += logLine.Content
					logLines[len(logLines)-1].ByteEnd += int64(len(logLine.Content))
				}
				continue
			}
		}

		logLine.ByteStart = byteStart
		logLine.ByteContentStart = byteStart + int64(len(line)-len(logLine.Content))
		logLine.ByteEnd = byteStart + int64(len(line))

		// Generate unique ID that can be used to reference this line
		logLine.UUID = uuid.NewV4()

		logLines = append(logLines, logLine)
	}

	newLogLines, newSamples := AnalyzeLogLines(logLines)
	return newLogLines, newSamples
}
