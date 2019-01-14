package logs

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	uuid "github.com/satori/go.uuid"
)

// AnalyzeStreamInGroups - Splits log lines into those that are ready, and those that aren't
func AnalyzeStreamInGroups(logLines []state.LogLine) (state.LogState, state.LogFile, []state.LogLine, error) {
	var readyLogLines []state.LogLine
	var tooFreshLogLines []state.LogLine
	var stitchedLogLines []state.LogLine

	// Submit all logLines that are older than 3 seconds
	var now time.Time
	now = time.Now()

	// Always stitch together log lines ahead of time that are missing level and PID
	// - this is mostly to support the output of the Postgres logging collector to files
	for _, logLine := range logLines {
		if logLine.LogLevel != pganalyze_collector.LogLineInformation_UNKNOWN || logLine.BackendPid != 0 {
			stitchedLogLines = append(stitchedLogLines, logLine)
		} else if len(stitchedLogLines) > 0 {
			stitchedLogLines[len(stitchedLogLines)-1].Content += " " + logLine.Content
		}
	}

	for _, logLine := range stitchedLogLines {
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
		return state.LogState{}, state.LogFile{}, tooFreshLogLines, nil
	}

	// Setup temporary file that will be used for encryption
	var logFile state.LogFile
	var err error
	logFile.UUID = uuid.NewV4()
	logFile.TmpFile, err = ioutil.TempFile("", "")
	if err != nil {
		return state.LogState{}, state.LogFile{}, logLines, fmt.Errorf("Could not allocate tempfile for logs: %s", err)
	}

	logState := state.LogState{CollectedAt: time.Now()}

	currentByteStart := int64(0)
	for idx, logLine := range readyLogLines {
		_, err = logFile.TmpFile.WriteString(logLine.Content)
		if err != nil {
			logFile.Cleanup()
			return state.LogState{}, state.LogFile{}, logLines, err
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

	return logState, logFile, tooFreshLogLines, nil
}
