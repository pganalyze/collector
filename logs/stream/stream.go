package stream

import (
	"fmt"
	"io/ioutil"
	"sort"
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

// This file handles stream-based log collection. Currently this is used in two cases:
// (1) Self-managed VMs (local log tail)
// (2) Heroku Postgres (network log drain)
//
// Self-managed VM log data looks like this:
// - Correctly ordered (we never have to sort the line for a specific PID earlier than another line)
// - Subsequent lines will be missing PID data (since the log_line_prefix doesn't get output again)
// - Data from different PIDs might be mixed, so we need to split data up by PID before analysis
//
// Heroku log data looks like this:
// - Not correctly ordered (lines from logplex may arrive in any order)
// - Always have PID data for each line
// - Always have the log line number, allowing association of related log lines

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
					// Self-managed case - we always stitch unknown log levels to the prior line
					readyLogLines = append(readyLogLines, logLine)
				} else {
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
func handleLogAnalysis(readyLogLines []state.LogLine) ([]state.LogLine, []state.PostgresQuerySample) {
	// Ensure that log lines that span multiple lines are already concated together before passing them to analyze
	// Split log lines by backend to ensure we have the right context
	backendLogLines := make(map[int32][]state.LogLine)

	for _, logLine := range readyLogLines {
		backendLogLines[logLine.BackendPid] = append(backendLogLines[logLine.BackendPid], logLine)
	}

	var logLinesOut []state.LogLine
	var querySamples []state.PostgresQuerySample

	for _, logLines := range backendLogLines {
		var analyzableLogLines []state.LogLine
		for _, logLine := range logLines {
			if logLine.LogLevel != pganalyze_collector.LogLineInformation_UNKNOWN {
				analyzableLogLines = append(analyzableLogLines, logLine)
			} else if len(analyzableLogLines) > 0 {
				analyzableLogLines[len(analyzableLogLines)-1].Content += logLine.Content
				analyzableLogLines[len(analyzableLogLines)-1].ByteEnd += int64(len(logLine.Content))
			}
		}

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

// AnalyzeStreamInGroups - Takes in a set of parsed log lines and analyses the
// lines that are ready, and returns the rest
//
// The caller is expected to keep a repository of "tooFreshLogLines" that they
// can send back in again in the next call, combined with new lines received
func AnalyzeStreamInGroups(logLines []state.LogLine) (state.LogState, state.LogFile, []state.LogLine, error) {
	// Pre-Sort by PID, log line number and occurred at timestamp
	// 
	// Its important we do this early, to support out-of-order receipt of log lines,
	// up to the freshness threshold used in the next function call (3 seconds)
	sort.Slice(logLines, func(i, j int) bool {
		if logLines[i].BackendPid != logLines[j].BackendPid {
			return logLines[i].BackendPid < logLines[j].BackendPid
		}
		if logLines[i].LogLineNumber != logLines[j].LogLineNumber {
			return logLines[i].LogLineNumber < logLines[j].LogLineNumber
		}
		return logLines[i].OccurredAt.Unix() < logLines[j].OccurredAt.Unix()
	})

	readyLogLines, tooFreshLogLines := findReadyLogLines(logLines, 3*time.Second)
	if len(readyLogLines) == 0 {
		return state.LogState{}, state.LogFile{}, tooFreshLogLines, nil
	}

	logFile, err := writeTmpLogFile(readyLogLines)
	if err != nil {
		return state.LogState{}, state.LogFile{}, logLines, err
	}

	logState := state.LogState{CollectedAt: time.Now()}
	logFile.LogLines, logState.QuerySamples = handleLogAnalysis(readyLogLines)

	return logState, logFile, tooFreshLogLines, nil
}

// ProcessLogStream - Accepts one or more log lines to be analyzed and processed
//
// Note that this returns the lines that were not processed, based on the
// time-based buffering logic. These lines should be passed in again with
// the next call.
//
// The caller is not expected to do any special time-based buffering themselves.
func ProcessLogStream(server state.Server, logLines []state.LogLine, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger, logTestSucceeded chan<- bool) []state.LogLine {
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

	if server.Config.EnableLogExplain {
		db, err := postgres.EstablishConnection(server, prefixedLogger, globalCollectionOpts, "")
		if err == nil {
			logState.QuerySamples = postgres.RunExplain(db, server.Config.GetDbName(), logState.QuerySamples)
			db.Close()
		}
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
		for _, logLine := range logFile.LogLines {
			if logLine.Classification == pganalyze_collector.LogLineInformation_PGA_COLLECTOR_IDENTIFY &&
				logLine.Details["config_section"] == server.Config.SectionName {
				logTestSucceeded <- true
			}
		}
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
		prefixedLogger.PrintVerbose("Log collection disabled from server, skipping")
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
