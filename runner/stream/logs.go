package stream

import (
	"io/ioutil"

	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func ProcessLogs(server state.Server, logLines []state.LogLine, globalCollectionOpts state.CollectionOpts, prefixedLogger *util.Logger, logTestSucceeded chan<- bool) []state.LogLine {
	logState, logFile, tooFreshLogLines, err := logs.AnalyzeStreamInGroups(logLines)
	if err != nil {
		prefixedLogger.PrintError("%s", err)
		return tooFreshLogLines
	}

	// Nothing to send, so just skip getting the grant and other work
	if len(logFile.LogLines) == 0 && len(logState.QuerySamples) == 0 {
		logState.Cleanup()
		return tooFreshLogLines
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
