package stream

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// This file handles stream-based log collection. This is used in multiple cases:
// (1) Self-managed VMs (local log tail or syslog)
// (2) Heroku Postgres (network log drain)
// (3) Google Cloud SQL (GCP Pub/Sub)
// (4) Azure Database for PostgreSQL (Azure Event Hub)
//
// Self-managed VM log data looks like this:
// - Correctly ordered (we never have to sort the line for a specific PID earlier than another line)
// - Data from different PIDs might be mixed, so we need to split data up by PID before analysis
// - Additionally, for log tails (but not syslog receiver):
//   - Subsequent lines will be missing PID data (since the log_line_prefix doesn't get output again)
//   - Subsequent lines will be missing timestamp data (only lines with log_line_prefix have a timestamp)
//
// Heroku log data looks like this:
// - Not correctly ordered (lines from logplex may arrive in any order)
// - Always has PID data for each line
// - Always has the log line number, allowing association of related log lines
//
// Google Cloud SQL log data looks like this:
// - Not correctly ordered (lines from Pub/Sub may arrive in any order)
// - All lines have a timestamp (taken from the Timestamp field in the original Cloud Logging message)
// - First line of a message always has log line number (due to fixed log_line_prefix)
// - Log events split across multiple log messages only have the PID and line number in the first message,
//   may arrive out of order, but have a timestamp that is correctly ordering each part of the log event
// - Split log events are rare (as of Sept 14, 2021 - see https://cloud.google.com/sql/docs/release-notes#September_14_2021),
//   but can still happen if the log data is too big (cutoff appears to be around 1000-2000 lines, or ~100kb of data)

const InvalidPid int32 = -1
const UnknownPid int32 = 0

type LogLineReadiness int

const (
	LogLineDefer LogLineReadiness = iota
	LogLineReady
	LogLineDiscard
)

func determineLogLineReadiness(logLine state.LogLine, threshold time.Duration, now time.Time, lastReadyMainLogLinePid int32, lastReadyLogLineWithPrefixPid int32) LogLineReadiness {
	// The easy case: We have a log line with a log_line_prefix, and only need
	// to check if we are ready to send or whether there could still be
	// subsequent lines that will show up within the threshold
	if logLine.LogLevel != pganalyze_collector.LogLineInformation_UNKNOWN {
		if now.Sub(logLine.CollectedAt) > threshold || (isAdditionalLineLevel(logLine.LogLevel) && logLine.BackendPid == lastReadyMainLogLinePid) {
			return LogLineReady
		}

		return LogLineDefer
	}

	// Part of a multi-line log line, where we don't have a log_line_prefix,
	// but we have a PID, and so can make some assumptions of where this
	// log line might fit. This is the case for Heroku and syslog receivers.
	if logLine.BackendPid != UnknownPid {
		if logLine.BackendPid == lastReadyLogLineWithPrefixPid {
			return LogLineReady
		}

		return LogLineDefer
	}

	// Part of a multi-line log line, where we don't have a log_line_prefix,
	// or any other contextual information!
	//
	// This is the case for self-managed servers where we tail a log file,
	// and so we can reasonably assume that subsequent unidentifiable lines
	// directly belong to the line we saw right before.
	//
	// There are edge cases where we discard lines here, since we want to avoid
	// keeping lines forever that we could never associate to anything.
	if lastReadyLogLineWithPrefixPid != InvalidPid {
		return LogLineReady
	}
	if now.Sub(logLine.CollectedAt) > threshold {
		return LogLineDiscard
	}
	return LogLineDefer
}

// findReadyLogLines - Splits log lines into those that are ready, and those that aren't
func findReadyLogLines(logLines []state.LogLine, now time.Time, threshold time.Duration) ([]state.LogLine, []state.LogLine) {
	var readyLogLines []state.LogLine
	var tooFreshLogLines []state.LogLine
	var lastReadyMainLogLinePid int32 = InvalidPid
	var lastReadyLogLineWithPrefixPid int32 = InvalidPid

	for _, logLine := range logLines {
		action := determineLogLineReadiness(logLine, threshold, now, lastReadyMainLogLinePid, lastReadyLogLineWithPrefixPid)

		switch action {
		case LogLineDefer:
			// Keep for next run
			tooFreshLogLines = append(tooFreshLogLines, logLine)
		case LogLineReady:
			// Send to pganalyze for processing
			readyLogLines = append(readyLogLines, logLine)
		case LogLineDiscard:
			// Throw away this line
		}

		if logLine.LogLevel != pganalyze_collector.LogLineInformation_UNKNOWN {
			if action == LogLineReady && isMainLogLevel(logLine.LogLevel) {
				lastReadyMainLogLinePid = logLine.BackendPid
			} else if action == LogLineReady && isAdditionalLineLevel(logLine.LogLevel) {
				// Keep prior main log line PID
			} else {
				lastReadyMainLogLinePid = InvalidPid
			}

			if action == LogLineReady {
				lastReadyLogLineWithPrefixPid = logLine.BackendPid
			} else {
				lastReadyLogLineWithPrefixPid = InvalidPid
			}
		}
	}

	return readyLogLines, tooFreshLogLines
}

/* Level lists current as of Postgres 14.1 */
func isMainLogLevel(str pganalyze_collector.LogLineInformation_LogLevel) bool {
	switch str {
	case pganalyze_collector.LogLineInformation_DEBUG,
		pganalyze_collector.LogLineInformation_INFO,
		pganalyze_collector.LogLineInformation_NOTICE,
		pganalyze_collector.LogLineInformation_WARNING,
		pganalyze_collector.LogLineInformation_ERROR,
		pganalyze_collector.LogLineInformation_LOG,
		pganalyze_collector.LogLineInformation_FATAL,
		pganalyze_collector.LogLineInformation_PANIC:
		return true
	}
	return false
}
func isAdditionalLineLevel(str pganalyze_collector.LogLineInformation_LogLevel) bool {
	switch str {
	case pganalyze_collector.LogLineInformation_DETAIL,
		pganalyze_collector.LogLineInformation_HINT,
		pganalyze_collector.LogLineInformation_CONTEXT,
		pganalyze_collector.LogLineInformation_STATEMENT,
		pganalyze_collector.LogLineInformation_QUERY:
		return true
	}
	return false
}

func createLogFile(readyLogLines []state.LogLine, logger *util.Logger) (state.LogFile, error) {
	logFile, err := state.NewLogFile("")
	if err != nil {
		return state.LogFile{}, fmt.Errorf("could not initialize log file: %s", err)
	}

	currentByteStart := int64(0)
	for idx, logLine := range readyLogLines {
		logLine.ByteStart = currentByteStart
		logLine.ByteContentStart = currentByteStart
		logLine.ByteEnd = currentByteStart + int64(len(logLine.Content))
		readyLogLines[idx] = logLine
		currentByteStart += int64(len(logLine.Content))
	}

	return logFile, nil
}

// handleLogAnalysis - Performs log analysis on submitted log lines on a per-backend basis
func handleLogAnalysis(analyzableLogLines []state.LogLine) ([]state.LogLine, []state.PostgresQuerySample) {
	// Split log lines by backend to ensure we have the right context
	backendLogLines := make(map[int32][]state.LogLine)

	for _, logLine := range analyzableLogLines {
		backendLogLines[logLine.BackendPid] = append(backendLogLines[logLine.BackendPid], logLine)
	}

	var logLinesOut []state.LogLine
	var querySamples []state.PostgresQuerySample

	for _, analyzableLogLines := range backendLogLines {
		backendLogLinesOut, backendSamples := logs.AnalyzeBackendLogLines(analyzableLogLines)
		for _, logLine := range backendLogLinesOut {
			logLinesOut = append(logLinesOut, logLine)
		}
		for _, sample := range backendSamples {
			querySamples = append(querySamples, sample)
		}
	}

	return logLinesOut, querySamples
}

func stitchLogLines(readyLogLines []state.LogLine) (analyzableLogLines []state.LogLine) {
	var linesToAppend []int
	var linesToAppendLenSum int
	var b strings.Builder
	for idx, logLine := range readyLogLines {
		if logLine.LogLevel == pganalyze_collector.LogLineInformation_UNKNOWN {
			if len(analyzableLogLines) > 0 {
				linesToAppend = append(linesToAppend, idx)
				linesToAppendLenSum = linesToAppendLenSum + len(logLine.Content)
			}
		} else {
			if linesToAppendLenSum > 0 {
				b.Grow(linesToAppendLenSum)
				for _, logIdx := range linesToAppend {
					b.WriteString(readyLogLines[logIdx].Content)
				}
				analyzableLogLines[len(analyzableLogLines)-1].Content += b.String()
				b.Reset()
				linesToAppend = nil
				linesToAppendLenSum = 0
			}
			analyzableLogLines = append(analyzableLogLines, logLine)
		}
	}
	if linesToAppendLenSum > 0 {
		b.Grow(linesToAppendLenSum)
		for _, logIdx := range linesToAppend {
			b.WriteString(readyLogLines[logIdx].Content)
		}
		analyzableLogLines[len(analyzableLogLines)-1].Content += b.String()
		b.Reset()
	}
	return
}

const StreamReadyThreshold time.Duration = 3 * time.Second

// AnalyzeStreamInGroups - Takes in a set of parsed log lines and analyses the
// lines that are ready, and returns the rest
//
// The caller is expected to keep a repository of "tooFreshLogLines" that they
// can send back in again in the next call, combined with new lines received
func AnalyzeStreamInGroups(logLines []state.LogLine, now time.Time, server *state.Server, logger *util.Logger) (state.TransientLogState, state.LogFile, []state.LogLine, error) {
	// Pre-Sort by PID, log line number and occurred at timestamp
	//
	// Its important we do this early, to support out-of-order receipt of log lines,
	// up to the freshness threshold used for findReadyLogLines (3 seconds)
	allLinesHaveBackendPid := true
	allLinesHaveLogLineNumber := true
	allLinesHaveLogLineNumberChunk := true
	allLinesHaveOccurredAt := true
	for _, logLine := range logLines {
		if logLine.BackendPid == UnknownPid {
			allLinesHaveBackendPid = false
		}
		if logLine.LogLineNumber == 0 {
			allLinesHaveLogLineNumber = false
		}
		if logLine.LogLineNumberChunk == 0 {
			allLinesHaveLogLineNumberChunk = false
		}
		if logLine.OccurredAt.IsZero() {
			allLinesHaveOccurredAt = false
		}
	}
	sort.SliceStable(logLines, func(i, j int) bool {
		if allLinesHaveBackendPid && logLines[i].BackendPid != logLines[j].BackendPid {
			return logLines[i].BackendPid < logLines[j].BackendPid
		}
		if allLinesHaveLogLineNumber && logLines[i].LogLineNumber != logLines[j].LogLineNumber {
			return logLines[i].LogLineNumber < logLines[j].LogLineNumber
		}
		if allLinesHaveLogLineNumberChunk && logLines[i].LogLineNumberChunk != logLines[j].LogLineNumberChunk {
			return logLines[i].LogLineNumberChunk < logLines[j].LogLineNumberChunk
		}
		if allLinesHaveOccurredAt {
			return logLines[i].OccurredAt.Sub(logLines[j].OccurredAt) < 0
		}
		return false // Keep initial order
	})

	readyLogLines, tooFreshLogLines := findReadyLogLines(logLines, now, StreamReadyThreshold)
	if len(readyLogLines) == 0 {
		return state.TransientLogState{}, state.LogFile{}, tooFreshLogLines, nil
	}

	// Ensure that log lines that span multiple lines are already concated together before passing them to analyze
	//
	// Since we already sorted by PID earlier, it is safe for us to concatenate lines before grouping. In fact,
	// this is required for cases where unknown log lines don't have PIDs associated
	stitchedLogLines := stitchLogLines(readyLogLines)

	var analyzableLogLines []state.LogLine
	for _, logLine := range stitchedLogLines {
		if !server.IgnoreLogLine(logLine.Content) {
			analyzableLogLines = append(analyzableLogLines, logLine)
		}
	}

	logFile, err := createLogFile(analyzableLogLines, logger)
	if err != nil {
		return state.TransientLogState{}, state.LogFile{}, logLines, err
	}

	logState := state.TransientLogState{CollectedAt: now}
	logFile.LogLines, logState.QuerySamples = handleLogAnalysis(analyzableLogLines)

	return logState, logFile, tooFreshLogLines, nil
}

// Log test functions used to verify whether a stream works

// LogTestCollectorIdentify - Checks for the special "pganalyze-collector-identify:" event
// (used on log pipelines that forward messages under than 10 seconds)
func LogTestCollectorIdentify(server *state.Server, logFile state.LogFile, logTestSucceeded chan<- bool) {
	for _, logLine := range logFile.LogLines {
		if logLine.Classification == pganalyze_collector.LogLineInformation_PGA_COLLECTOR_IDENTIFY &&
			logLine.Details["config_section"] == server.Config.SectionName {
			logTestSucceeded <- true
		}
	}
}

// LogTestAnyEvent - Checks for any log message
// (used on log pipelines that take longer than 10 seconds, e.g. Azure Event Hub)
func LogTestAnyEvent(server *state.Server, logFile state.LogFile, logTestSucceeded chan<- bool) {
	logTestSucceeded <- true
}

// LogTestNone - Don't confirm the log test
func LogTestNone(server *state.Server, logFile state.LogFile, logTestSucceeded chan<- bool) {
}
