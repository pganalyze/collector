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
const LogPrefixCustom1 string = "%m [%p][%v] : [%l-1] %q[app=%a] "
const LogPrefixCustom2 string = "%t [%p-%l] %q%u@%d "

// Every one of these regexps should produce exactly one matching group
var TimeRegexp = `(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)? \w+)` // %t or %m
var IpAndPortRegexp = `([\d:.]+\(\d+\))?`                              // %r
var PidRegexp = `(\d+)`                                                // %p
var UserRegexp = `(\S*)`                                               // %u
var DbRegexp = `(\S*)`                                                 // %d
var AppRegexp = `(\S*)`                                                // %a
var VirtualTxRegexp = `(\d+/\d+)?`                                     // %v
var LogLineCounterRegexp = `(\d+)`                                     // %l
// Missing:
// - %h (host without port)
// - %n (unix timestamp)
// - %i (command tag)
// - %e (SQLSTATE)
// - %c (session ID)
// - %s (process start timestamp)
// - %x (transaction ID)

var LevelAndContentRegexp = `(\w+):\s+(.*\n?)$`
var LogPrefixAmazonRdsRegxp = regexp.MustCompile(`^` + TimeRegexp + `:` + IpAndPortRegexp + `:` + UserRegexp + `@` + DbRegexp + `:\[` + PidRegexp + `\]:` + LevelAndContentRegexp)
var LogPrefixCustom1Regexp = regexp.MustCompile(`^` + TimeRegexp + ` \[` + PidRegexp + `\]\[` + VirtualTxRegexp + `\] : \[` + LogLineCounterRegexp + `-1\] (?:\[app=` + AppRegexp + `\] )?` + LevelAndContentRegexp)
var LogPrefixCustom2Regexp = regexp.MustCompile(`^` + TimeRegexp + ` \[` + PidRegexp + `-` + LogLineCounterRegexp + `\] ` + `(?:` + UserRegexp + `@` + DbRegexp + ` )?` + LevelAndContentRegexp)

func parseLogLineWithPrefix(prefix string, line string) (logLine state.LogLine, ok bool) {
	var timePart, userPart, dbPart, appPart, pidPart, levelPart, contentPart string

	if prefix == "" {
		if LogPrefixAmazonRdsRegxp.MatchString(line) {
			prefix = LogPrefixAmazonRds
		} else if LogPrefixCustom1Regexp.MatchString(line) {
			prefix = LogPrefixCustom1
		} else if LogPrefixCustom2Regexp.MatchString(line) {
			prefix = LogPrefixCustom2
		}
	}

	switch prefix {
	case LogPrefixAmazonRds: // "%t:%r:%u@%d:[%p]:"
		parts := LogPrefixAmazonRdsRegxp.FindStringSubmatch(line)
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
	case LogPrefixCustom1: // "%m [%p][%v] : [%l-1] %q[app=%a] "
		parts := LogPrefixCustom1Regexp.FindStringSubmatch(line)
		if len(parts) == 0 {
			return
		}
		timePart = parts[1]
		pidPart = parts[2]
		// skip %v (virtual TX)
		// skip %l (log line counter)
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
		// skip %l (log line counter)
		userPart = parts[4]
		dbPart = parts[5]
		levelPart = parts[6]
		contentPart = parts[7]
	default:
		return
	}

	var err error
	logLine.OccurredAt, err = time.Parse("2006-01-02 15:04:05 MST", timePart)
	if err != nil {
		ok = false
		return
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

	backendPid, _ := strconv.Atoi(pidPart)
	logLine.BackendPid = int32(backendPid)
	logLine.LogLevel = pganalyze_collector.LogLineInformation_LogLevel(pganalyze_collector.LogLineInformation_LogLevel_value[levelPart])
	logLine.Content = contentPart

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

		logLine, ok := parseLogLineWithPrefix("", line)
		if !ok {
			// Assume that a parsing error in a follow-on line means that we actually
			// got additional data for the previous line
			if len(logLines) > 0 {
				logLines[len(logLines)-1].Content += line
				logLines[len(logLines)-1].ByteEnd += int64(len(line))
			}
			continue
		}

		// Ignore loglines which are outside our time window
		if logLine.OccurredAt.Before(linesNewerThan) {
			continue
		}

		logLine.ByteStart = byteStart
		logLine.ByteContentStart = byteStart + int64(len(line)-len(logLine.Content))
		logLine.ByteEnd = byteStart + int64(len(line)) - 1

		// Generate unique ID that can be used to reference this line
		logLine.UUID = uuid.NewV4()

		logLines = append(logLines, logLine)
	}

	newLogLines, newSamples := AnalyzeLogLines(logLines)
	return newLogLines, newSamples, currentByteStart
}
