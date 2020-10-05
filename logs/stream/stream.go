package stream

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

// This file handles stream-based log collection. Currently this is used in three cases:
// (1) Self-managed VMs (local log tail)
// (2) Heroku Postgres (network log drain)
// (3) Google Cloud SQL (GCP Pub/Sub)
//
// Self-managed VM log data looks like this:
// - Correctly ordered (we never have to sort the line for a specific PID earlier than another line)
// - Subsequent lines will be missing PID data (since the log_line_prefix doesn't get output again)
// - Data from different PIDs might be mixed, so we need to split data up by PID before analysis
// - Only lines with log_line_prefix have a timestamp
//
// Heroku log data looks like this:
// - Not correctly ordered (lines from logplex may arrive in any order)
// - Always has PID data for each line
// - Always has the log line number, allowing association of related log lines
//
// Google Cloud SQL log data looks like this:
// - Not correctly ordered (lines from Pub/Sub may arrive in any order)
// - All lines have a timestamp
// - First line of a message always has log line number (due to fixed log_line_prefix)
// - Multi-line messages only have the PID and line number in the first message

// findReadyLogLines - Splits log lines into those that are ready, and those that aren't
func findReadyLogLines(logLines []state.LogLine, threshold time.Duration) ([]state.LogLine, []state.LogLine) {
	var readyLogLines []state.LogLine
	var tooFreshLogLines []state.LogLine
	var lastPrimaryLogLine *state.LogLine

	now := time.Now()

	for _, logLine := range logLines {
		if logLine.LogLevel == pganalyze_collector.LogLineInformation_UNKNOWN {
			if logLine.BackendPid != 0 {
				// Heroku case (Unknown log lines have PIDs assigned)
				if lastPrimaryLogLine != nil && logLine.BackendPid == lastPrimaryLogLine.BackendPid {
					readyLogLines = append(readyLogLines, logLine)
				} else {
					tooFreshLogLines = append(tooFreshLogLines, logLine)
				}
			} else {
				if lastPrimaryLogLine != nil {
					// Self-managed and Google Cloud SQL case - we always stitch unknown log levels to the prior line
					readyLogLines = append(readyLogLines, logLine)
				} else if now.Sub(logLine.CollectedAt) <= threshold {
					tooFreshLogLines = append(tooFreshLogLines, logLine)
				}
			}
		} else {
			if now.Sub(logLine.CollectedAt) > threshold {
				readyLogLines = append(readyLogLines, logLine)
				lastPrimaryLogLine = &logLine
			} else {
				tooFreshLogLines = append(tooFreshLogLines, logLine)
				lastPrimaryLogLine = nil
			}
		}
	}

	return readyLogLines, tooFreshLogLines
}

// writeTmpLogFile - Setup temporary file that will be used for encryption
func writeTmpLogFile(readyLogLines []state.LogLine) (state.LogFile, error) {
	var logFile state.LogFile
	var err error
	logFile.UUID = uuid.NewV4()
	logFile.TmpFile, err = ioutil.TempFile("", "")
	if err != nil {
		return state.LogFile{}, fmt.Errorf("Could not allocate tempfile for logs: %s", err)
	}

	currentByteStart := int64(0)
	for idx, logLine := range readyLogLines {
		_, err = logFile.TmpFile.WriteString(logLine.Content)
		if err != nil {
			logFile.Cleanup()
			return logFile, err
		}
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

// AnalyzeStreamInGroups - Takes in a set of parsed log lines and analyses the
// lines that are ready, and returns the rest
//
// The caller is expected to keep a repository of "tooFreshLogLines" that they
// can send back in again in the next call, combined with new lines received
func AnalyzeStreamInGroups(logLines []state.LogLine) (state.TransientLogState, state.LogFile, []state.LogLine, error) {
	// Pre-Sort by PID, log line number and occurred at timestamp
	//
	// Its important we do this early, to support out-of-order receipt of log lines,
	// up to the freshness threshold used in the next function call (3 seconds)
	sort.SliceStable(logLines, func(i, j int) bool {
		if logLines[i].BackendPid != 0 && logLines[j].BackendPid != 0 && logLines[i].BackendPid != logLines[j].BackendPid {
			return logLines[i].BackendPid < logLines[j].BackendPid
		}
		if logLines[i].LogLineNumber != 0 && logLines[j].LogLineNumber != 0 && logLines[i].LogLineNumber != logLines[j].LogLineNumber {
			return logLines[i].LogLineNumber < logLines[j].LogLineNumber
		}
		if !logLines[i].OccurredAt.IsZero() && !logLines[j].OccurredAt.IsZero() {
			return logLines[i].OccurredAt.Sub(logLines[j].OccurredAt) < 0
		}
		return false // Keep initial order
	})

	readyLogLines, tooFreshLogLines := findReadyLogLines(logLines, 3*time.Second)
	if len(readyLogLines) == 0 {
		return state.TransientLogState{}, state.LogFile{}, tooFreshLogLines, nil
	}

	// Ensure that log lines that span multiple lines are already concated together before passing them to analyze
	//
	// Since we already sorted by PID earlier, it is safe for us to concatenate lines before grouping. In fact,
	// this is required for cases where unknown log lines don't have PIDs associated
	analyzableLogLines := stitchLogLines(readyLogLines)

	logFile, err := writeTmpLogFile(analyzableLogLines)
	if err != nil {
		return state.TransientLogState{}, state.LogFile{}, logLines, err
	}

	logState := state.TransientLogState{CollectedAt: time.Now()}
	logFile.LogLines, logState.QuerySamples = handleLogAnalysis(analyzableLogLines)

	return logState, logFile, tooFreshLogLines, nil
}

// Log test functions used to verify whether a stream works

// LogTestCollectorIdentify - Checks for the special "pganalyze-collector-identify:" event
// (used on log pipelines that forward messages under than 10 seconds)
func LogTestCollectorIdentify(server state.Server, logFile state.LogFile, logTestSucceeded chan<- bool) {
	for _, logLine := range logFile.LogLines {
		if logLine.Classification == pganalyze_collector.LogLineInformation_PGA_COLLECTOR_IDENTIFY &&
			logLine.Details["config_section"] == server.Config.SectionName {
			logTestSucceeded <- true
		}
	}
}

// LogTestAnyEvent - Checks for any log message
// (used on log pipelines that take longer than 10 seconds, e.g. Azure Event Hub)
func LogTestAnyEvent(server state.Server, logFile state.LogFile, logTestSucceeded chan<- bool) {
	logTestSucceeded <- true
}

// LogTestNone - Don't confirm the log test
func LogTestNone(server state.Server, logFile state.LogFile, logTestSucceeded chan<- bool) {
}

// ProcessLogStream - Accepts one or more log lines to be analyzed and processed
//
// Note that this returns the lines that were not processed, based on the
// time-based buffering logic. These lines should be passed in again with
// the next call.
//
// The caller is not expected to do any special time-based buffering themselves.
func ProcessLogStream(server state.Server, logLines []state.LogLine, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger, logTestSucceeded chan<- bool, logTestFunc func(s state.Server, lf state.LogFile, lt chan<- bool)) []state.LogLine {
	logState, logFile, tooFreshLogLines, err := AnalyzeStreamInGroups(logLines)
	if err != nil {
		prefixedLogger.PrintError("%s", err)
		return tooFreshLogLines
	}

	// Nothing to send, so just skip getting the grant and other work
	if len(logFile.LogLines) == 0 && len(logState.QuerySamples) == 0 {
		logState.Cleanup()
		return tooFreshLogLines
	}

	if server.Config.EnableLogExplain && len(logState.QuerySamples) != 0 {
		logState.QuerySamples = postgres.RunExplain(server, logState.QuerySamples, globalCollectionOpts, prefixedLogger)
	}

	logState.LogFiles = []state.LogFile{logFile}

	if globalCollectionOpts.DebugLogs {
		prefixedLogger.PrintInfo("Would have sent log state:\n")
		content, _ := ioutil.ReadFile(logFile.TmpFile.Name())
		logs.PrintDebugInfo(string(content), logFile.LogLines, logState.QuerySamples)
		logState.Cleanup()
		return tooFreshLogLines
	}

	if globalCollectionOpts.TestRun {
		logTestFunc(server, logFile, logTestSucceeded)
		logState.Cleanup()
		return tooFreshLogLines
	}

	grant, err := grant.GetLogsGrant(server, globalCollectionOpts, prefixedLogger)
	if err != nil {
		prefixedLogger.PrintError("Could not get log grant: %s", err)
		logState.Cleanup()
		return logLines // Retry
	}

	if !grant.Valid {
		prefixedLogger.PrintVerbose("Skipping log data: Feature not available on this pganalyze plan, or log data limit exceeded")
		logState.Cleanup()
		return tooFreshLogLines
	}

	err = output.UploadAndSendLogs(server, grant, globalCollectionOpts, prefixedLogger, logState)
	if err != nil {
		prefixedLogger.PrintError("Failed to upload/send logs: %s", err)
		logState.Cleanup()
		return logLines // Retry
	}

	logState.Cleanup()
	return tooFreshLogLines
}
