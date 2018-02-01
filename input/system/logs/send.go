package logs

import (
	"io/ioutil"
	"time"

	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

// Sends all log lines that are ready, and returns the one that are not ready yet
func AnalyzeInGroupsAndSend(server state.Server, logLines []state.LogLine, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger) []state.LogLine {
	var readyLogLines []state.LogLine
	var tooFreshLogLines []state.LogLine

	// Submit all logLines that are older than 3 seconds
	var now time.Time
	now = time.Now()

	for _, logLine := range logLines {
		// TODO: The intent here is to wait 3 seconds so we get follow-on log lines
		// (e.g. STATEMENT, HINT, DETAIL). This doesn't actually work, since we don't
		// peek into newer messages for these additional lines
		if now.Sub(logLine.CollectedAt) > 3*time.Second {
			readyLogLines = append(readyLogLines, logLine)
		} else {
			tooFreshLogLines = append(tooFreshLogLines, logLine)
		}
	}

	if len(readyLogLines) == 0 {
		return tooFreshLogLines
	}

	// Setup temporary file that will be used for encryption
	var logFile state.LogFile
	var err error
	logFile.UUID = uuid.NewV4()
	logFile.TmpFile, err = ioutil.TempFile("", "")
	if err != nil {
		prefixedLogger.PrintError("Could not allocate tempfile for logs: %s", err)
		return logLines
	}

	logState := state.LogState{CollectedAt: time.Now()}

	currentByteStart := int64(0)
	for idx, logLine := range readyLogLines {
		_, err = logFile.TmpFile.WriteString(logLine.Content)
		if err != nil {
			prefixedLogger.PrintError("%s", err)
			break
		}
		logLine.ByteStart = currentByteStart
		logLine.ByteContentStart = currentByteStart
		logLine.ByteEnd = currentByteStart + int64(len(logLine.Content)) - 1
		readyLogLines[idx] = logLine
		currentByteStart += int64(len(logLine.Content))
	}

	// Ensure that log lines that span multiple lines are already concated together before passing them to analyze
	// Split log lines by backend to ensure we have the right context
	backendLogLines := make(map[int32][]state.LogLine)

	for _, logLine := range readyLogLines {
		backendLogLines[logLine.BackendPid] = append(backendLogLines[logLine.BackendPid], logLine)
	}

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

		backendLogLinesOut, backendSamples := AnalyzeBackendLogLines(analyzableLogLines)
		for _, logLine := range backendLogLinesOut {
			logFile.LogLines = append(logFile.LogLines, logLine)
		}
		for _, sample := range backendSamples {
			logState.QuerySamples = append(logState.QuerySamples, sample)
		}
	}

	// Nothing to send, so just skip getting the grant and other work
	if len(logFile.LogLines) == 0 && len(logState.QuerySamples) == 0 {
		return tooFreshLogLines
	}

	logState.LogFiles = []state.LogFile{logFile}
	defer logState.Cleanup()

	if globalCollectionOpts.DebugLogs {
		prefixedLogger.PrintInfo("Would have sent log state:\n")
		content, _ := ioutil.ReadFile(logFile.TmpFile.Name())
		PrintDebugInfo(string(content), logFile.LogLines, logState.QuerySamples)
		return tooFreshLogLines
	}

	grant, err := grant.GetLogsGrant(server, globalCollectionOpts, prefixedLogger)
	if err != nil {
		prefixedLogger.PrintError("Could not get log grant: %s", err)
		return logLines // Retry
	}

	if !grant.Valid {
		prefixedLogger.PrintVerbose("Log collection disabled from server, skipping")
		return tooFreshLogLines
	}

	err = output.UploadAndSendLogs(server, grant, globalCollectionOpts, prefixedLogger, logState)
	if err != nil {
		prefixedLogger.PrintError("Failed to upload/send logs: %s", err)
		return logLines // Retry
	}

	return tooFreshLogLines
}
