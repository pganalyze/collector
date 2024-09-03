package logs

import (
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/pganalyze/collector/state"
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
const LogPrefixCustom11 string = "pid=%p,user=%u,db=%d,app=%a,client=%h "
const LogPrefixCustom12 string = "user=%u,db=%d,app=%a,client=%h "
const LogPrefixCustom13 string = "%p-%s-%c-%l-%h-%u-%d-%m "
const LogPrefixCustom14 string = "%m [%p][%b][%v][%x] %q[user=%u,db=%d,app=%a] "
const LogPrefixCustom15 string = "%m [%p] %q%u@%d "
const LogPrefixCustom16 string = "%t [%p] %q%u@%d %h "
const LogPrefixSimple string = "%m [%p] "
const LogPrefixHeroku1 string = " sql_error_code = %e "
const LogPrefixHeroku2 string = ` sql_error_code = %e time_ms = "%m" pid="%p" proc_start_time="%s" session_id="%c" vtid="%v" tid="%x" log_line="%l" %qdatabase="%d" connection_source="%r" user="%u" application_name="%a" `

// Used only to recognize the Heroku hobby tier log_line_prefix to give a warning (logs are not supported
// on hobby tier) and avoid errors during prefix check; logs with this prefix are never actually received
const LogPrefixHerokuHobbyTier string = " database = %d connection_source = %r sql_error_code = %e "
const LogPrefixEmpty string = ""

var RecommendedPrefixIdx = 4

var SupportedPrefixes = []string{
	LogPrefixAmazonRds, LogPrefixAzure, LogPrefixCustom1, LogPrefixCustom2,
	LogPrefixCustom3, LogPrefixCustom4, LogPrefixCustom5, LogPrefixCustom6,
	LogPrefixCustom7, LogPrefixCustom8, LogPrefixCustom9, LogPrefixCustom10,
	LogPrefixCustom11, LogPrefixCustom12, LogPrefixCustom13, LogPrefixCustom14,
	LogPrefixCustom15, LogPrefixCustom16,
	LogPrefixSimple, LogPrefixHeroku1, LogPrefixHeroku2, LogPrefixEmpty,
}

// Every one of these regexps should produce exactly one matching group
var TimeRegexp = `(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)? [\-+]?\w+)` // %t or %m (or %s)
var HostAndPortRegexp = `(.+(?:\(\d+\))?)?`                                  // %r
var PidRegexp = `(\d+)`                                                      // %p
var UserRegexp = `(\S*)`                                                     // %u
var DbRegexp = `(\S*)`                                                       // %d
var AppBeforeSpaceRegexp = `(\S*)`                                           // %a
var AppBeforeCommaRegexp = `([^,]*)`                                         // %a
var AppBeforeQuoteRegexp = `([^"]*)`                                         // %a
var AppInsideBracketsRegexp = `(\[unknown\]|[^,\]]*)`                        // %a
var HostRegexp = `(\S*)`                                                     // %h
var VirtualTxRegexp = `(\d+/\d+)?`                                           // %v
var LogLineCounterRegexp = `(\d+)`                                           // %l
var SqlstateRegexp = `(\w{5})`                                               // %e
var TransactionIdRegexp = `(\d+)`                                            // %x
var SessionIdRegexp = `(\w+\.\w+)`                                           // %c
var BackendTypeRegexp = `([\w ]+)`                                           // %b
// Missing:
// - %n (unix timestamp)
// - %i (command tag)

var LevelAndContentRegexp = `(\w+):\s+(.*\n?)$`
var LogPrefixAmazonRdsRegexp = regexp.MustCompile(`(?s)^` + TimeRegexp + `:` + HostAndPortRegexp + `:` + UserRegexp + `@` + DbRegexp + `:\[` + PidRegexp + `\]:` + LevelAndContentRegexp)
var LogPrefixAzureRegexp = regexp.MustCompile(`(?s)^` + TimeRegexp + `-` + SessionIdRegexp + `-` + LevelAndContentRegexp)
var LogPrefixCustom1Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]\[` + VirtualTxRegexp + `\] : \[` + LogLineCounterRegexp + `-1\] (?:\[app=` + AppInsideBracketsRegexp + `\] )?` + LevelAndContentRegexp)
var LogPrefixCustom2Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `-` + LogLineCounterRegexp + `\] ` + `(?:` + UserRegexp + `@` + DbRegexp + ` )?` + LevelAndContentRegexp)
var LogPrefixCustom3Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\] (?:\[user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppInsideBracketsRegexp + `\] )?` + LevelAndContentRegexp)
var LogPrefixCustom4Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\] (?:\[user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppBeforeCommaRegexp + `,host=` + HostRegexp + `\] )?` + LevelAndContentRegexp)
var LogPrefixCustom5Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] user=` + UserRegexp + `,db=` + DbRegexp + ` - PG-` + SqlstateRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom6Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppBeforeCommaRegexp + `,client=` + HostRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom7Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] \[trx_id=` + TransactionIdRegexp + `\] user=` + UserRegexp + `,db=` + DbRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom8Regexp = regexp.MustCompile(`(?s)^\[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] db=` + DbRegexp + `,user=` + UserRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom9Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` ` + HostAndPortRegexp + ` ` + UserRegexp + ` ` + AppBeforeSpaceRegexp + ` \[` + SessionIdRegexp + `\] \[` + PidRegexp + `\] ` + LevelAndContentRegexp)
var LogPrefixCustom10Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]: \[` + LogLineCounterRegexp + `-1\] db=` + DbRegexp + `,user=` + UserRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom11Regexp = regexp.MustCompile(`(?s)^pid=` + PidRegexp + `,user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppBeforeCommaRegexp + `,client=` + HostRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom12Regexp = regexp.MustCompile(`(?s)^user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppBeforeCommaRegexp + `,client=` + HostRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom13Regexp = regexp.MustCompile(`(?s)^` + PidRegexp + `-` + TimeRegexp + `-` + SessionIdRegexp + `-` + LogLineCounterRegexp + `-` + HostRegexp + `-` + UserRegexp + `-` + DbRegexp + `-` + TimeRegexp + ` ` + LevelAndContentRegexp)
var LogPrefixCustom14Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\]\[` + BackendTypeRegexp + `\]\[` + VirtualTxRegexp + `\]\[` + TransactionIdRegexp + `\] (?:\[user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppInsideBracketsRegexp + `\] )?` + LevelAndContentRegexp)
var LogPrefixCustom15Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\] ` + `(?:` + UserRegexp + `@` + DbRegexp + ` )?` + LevelAndContentRegexp)
var LogPrefixCustom16Regexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\] ` + `(?:` + UserRegexp + `@` + DbRegexp + ` ` + HostRegexp + ` )?` + LevelAndContentRegexp)
var LogPrefixSimpleRegexp = regexp.MustCompile(`(?s)^` + TimeRegexp + ` \[` + PidRegexp + `\] ` + LevelAndContentRegexp)
var LogPrefixNoTimestampUserDatabaseAppRegexp = regexp.MustCompile(`(?s)^\[user=` + UserRegexp + `,db=` + DbRegexp + `,app=` + AppInsideBracketsRegexp + `\] ` + LevelAndContentRegexp)
var LogPrefixHeroku1Regexp = regexp.MustCompile(`^ sql_error_code = ` + SqlstateRegexp + " " + LevelAndContentRegexp)
var LogPrefixHeroku2Regexp = regexp.MustCompile(`^ sql_error_code = ` + SqlstateRegexp + ` time_ms = "` + TimeRegexp + `" pid="` + PidRegexp + `" proc_start_time="` + TimeRegexp + `" session_id="` + SessionIdRegexp + `" vtid="` + VirtualTxRegexp + `" tid="` + TransactionIdRegexp + `" log_line="` + LogLineCounterRegexp + `" (?:database="` + DbRegexp + `" connection_source="` + HostAndPortRegexp + `" user="` + UserRegexp + `" application_name="` + AppBeforeQuoteRegexp + `" )?` + LevelAndContentRegexp)

var SyslogSequenceAndSplitRegexp = `(\[[\d-]+\])?`

var RsyslogLevelAndContentRegexp = `(?:(\w+):\s+)?(.*\n?)$`
var RsyslogTimeRegexp = `(\w+\s+\d+ \d{2}:\d{2}:\d{2})`
var RsyslogHostnameRegxp = `(\S+)`
var RsyslogProcessNameRegexp = `(\w+)`
var RsyslogRegexp = regexp.MustCompile(`^` + RsyslogTimeRegexp + ` ` + RsyslogHostnameRegxp + ` ` + RsyslogProcessNameRegexp + `\[` + PidRegexp + `\]: ` + SyslogSequenceAndSplitRegexp + ` ` + RsyslogLevelAndContentRegexp)

type LineReader interface {
	ReadString(delim byte) (string, error)
}

func ParseAndAnalyzeBuffer(logStream LineReader, linesNewerThan time.Time, server *state.Server) ([]state.LogLine, []state.PostgresQuerySample) {
	var logLines []state.LogLine
	var currentByteStart int64 = 0
	parser := server.GetLogParser()

	for {
		line, err := logStream.ReadString('\n')
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

		logLine, ok := parser.ParseLine(line)
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

		// Ignore loglines that are ignored server-wide (e.g. because they are
		// log_statement=all/log_duration=on lines). Note this intentionally
		// runs after multi-line log lines have been stitched together.
		if server.IgnoreLogLine(logLine.Content) {
			continue
		}

		logLine.ByteStart = byteStart
		logLine.ByteContentStart = byteStart + int64(len(line)-len(logLine.Content))
		logLine.ByteEnd = byteStart + int64(len(line))

		// Generate unique ID that can be used to reference this line
		logLine.UUID, err = uuid.NewV7()
		if err != nil {
			fmt.Printf("Failed to generate log line UUID: %s", err)
			continue
		}

		logLines = append(logLines, logLine)
	}

	newLogLines, newSamples := AnalyzeLogLines(logLines)
	return newLogLines, newSamples
}
