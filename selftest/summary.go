package selftest

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"

	"github.com/pganalyze/collector/state"
)

type SummaryVerbosity int

const (
	VerbosityTerse SummaryVerbosity = iota
	VerbosityNormal
	VerbosityVerbose
)

func PrintSummary(servers []*state.Server, verbose bool) {
	// There is a race condition here with standard log output; without this, some
	// Log Insights-related log lines can end up interspersed into the output.
	<-time.After(1 * time.Second)
	fmt.Fprintln(os.Stderr)

	var verbosity SummaryVerbosity
	if verbose {
		verbosity = VerbosityVerbose
	} else if len(servers) > 1 {
		verbosity = VerbosityTerse
	} else {
		verbosity = VerbosityNormal
	}

	for _, server := range servers {
		printServerTestSummary(server, verbosity)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr)
	}
}

var GreenCheck = color.New(color.FgHiGreen).Sprint("✓")
var YellowBang = color.New(color.FgHiYellow).Sprint("!")
var RedX = color.New(color.FgHiRed).Sprint("✗")
var GrayDash = color.New(color.FgWhite).Sprint("—")
var GrayQuestion = color.New(color.FgWhite).Sprint("?")

var ServerPrinter = color.New(color.FgCyan)
var URLPrinter = color.New(color.Underline)

func getStatusIcon(code state.CollectionStateCode) string {
	switch code {
	case state.CollectionStateUnchecked:
		return GrayQuestion
	case state.CollectionStateNotAvailable:
		return GrayDash
	case state.CollectionStateWarning:
		return YellowBang
	case state.CollectionStateError:
		return RedX
	case state.CollectionStateOkay:
		return GreenCheck
	default:
		return " "
	}
}

func getMaxDbNameLen(dbNames []string) int {
	var maxDbNameLen int
	for _, dbName := range dbNames {
		if thisNameLen := len(dbName); thisNameLen > maxDbNameLen {
			maxDbNameLen = thisNameLen
		}
	}
	return maxDbNameLen
}

func summarizeDbChecks(status *state.SelfTestResult, aspect state.DbCollectionAspect, isVerbose bool) (string, string) {
	dbNames := status.MonitoredDbs
	checks := status.AllDbAspectStatuses[aspect]
	var firstDb string
	if len(dbNames) > 0 {
		firstDb = dbNames[0]
	}
	var firstUncheckedDb string
	var anyChecked = false
	for _, dbName := range dbNames {
		if item, ok := checks[dbName]; ok && item.State != state.CollectionStateUnchecked {
			anyChecked = true
		} else if firstUncheckedDb == "" {
			firstUncheckedDb = dbName
		}
	}
	var firstErrorDb string
	var firstErrorDbMsg string
	var errorCount = 0
	for _, dbName := range dbNames {
		if item, ok := checks[dbName]; ok && item.State == state.CollectionStateError {
			errorCount++
			if firstErrorDb == "" {
				firstErrorDb = dbName
				firstErrorDbMsg = item.Msg
			}
		}
	}
	var allStateOkay bool = len(dbNames) > 0
	for _, dbName := range dbNames {
		if item, ok := checks[dbName]; !ok || item.State != state.CollectionStateOkay {
			allStateOkay = false
			break
		}
	}
	var icon string
	if allStateOkay {
		icon = GreenCheck
	} else {
		icon = RedX
	}

	var verboseHint string
	if !isVerbose {
		verboseHint = " (see details with --verbose)"
	}

	var summaryMsg string
	if len(checks) == 0 {
		summaryMsg = "could not check databases"
	} else if !anyChecked {
		if len(checks) > 1 {
			summaryMsg = fmt.Sprintf("could not check %s and %d other monitored database(s)%s", firstUncheckedDb, len(checks)-1, verboseHint)
		} else {
			summaryMsg = fmt.Sprintf("could not check database %s", firstUncheckedDb)
		}
	} else if errorCount > 1 {
		summaryMsg = fmt.Sprintf("found problems in %s and %d other monitored database(s)%s", firstErrorDb, errorCount-1, verboseHint)
	} else if errorCount > 0 {
		summaryMsg = fmt.Sprintf("found problem in database %s: %s", firstErrorDb, firstErrorDbMsg)
	} else if len(checks) > 1 {
		summaryMsg = fmt.Sprintf("ok in %s and %d other monitored database(s)%s", firstDb, len(checks)-1, verboseHint)
	} else {
		summaryMsg = fmt.Sprintf("ok in %s (no other databases are configured to be monitored)", firstDb)
	}

	return icon, summaryMsg
}

func printDbStatus(dbName string, dbStatus *state.CollectionAspectStatus, maxDbNameLen int) {
	var dbStatusIcon, dbMsg string
	if dbStatus == nil {
		dbStatusIcon = getStatusIcon(state.CollectionStateUnchecked)
	} else {
		dbStatusIcon = getStatusIcon(dbStatus.State)
		dbMsg = dbStatus.Msg
	}

	// Ensure that database names of different lengths line up
	dbNameFmtString := fmt.Sprintf("%%%ds", maxDbNameLen-len(dbName))
	fmt.Fprintf(os.Stderr, "\t\t%s %s:"+dbNameFmtString+"\t\t%s\n", dbStatusIcon, dbName, "", dbMsg)
}

func getAspectStatus(status *state.SelfTestResult, aspect state.CollectionAspect) (icon string, msg string) {
	aspectStatus := status.GetCollectionAspectStatus(aspect)
	if aspectStatus == nil {
		return getStatusIcon(state.CollectionStateUnchecked), ""
	}
	return getStatusIcon(aspectStatus.State), aspectStatus.Msg
}

func printServerTestSummary(s *state.Server, verbosity SummaryVerbosity) {
	verbose := verbosity == VerbosityVerbose
	config := s.Config
	status := s.SelfTest
	serverName := ServerPrinter.Sprintf(config.SectionName)
	fmt.Fprintf(os.Stderr, "Server %s:\n", serverName)
	fmt.Fprintln(os.Stderr)

	fmt.Fprintf(os.Stderr, "\t%s System Type:\t\t%s\n", GreenCheck, config.SystemType)
	fmt.Fprintf(os.Stderr, "\t%s System Scope:\t\t%s\n", GreenCheck, config.SystemScope)
	fmt.Fprintf(os.Stderr, "\t%s System ID:\t\t%s\n", GreenCheck, config.SystemID)

	if status.CollectionSuspended.Value {
		fmt.Fprintf(os.Stderr, "\t%s Collection suspended:\t%s\n", YellowBang, status.CollectionSuspended.Msg)
		return
	}

	apiIcon, apiMsg := getAspectStatus(status, state.CollectionAspectApiConnection)
	fmt.Fprintf(os.Stderr, "\t%s pganalyze connection:\t%s\n", apiIcon, apiMsg)
	telemetryIcon, telemetryMsg := getAspectStatus(status, state.CollectionAspectTelemetry)
	fmt.Fprintf(os.Stderr, "\t%s Collector telemetry:\t%s\n", telemetryIcon, telemetryMsg)

	if s.PGAnalyzeURL != "" {
		fmt.Fprintf(os.Stderr, "\t  View in pganalyze:\t%s\n", URLPrinter.Sprint(s.PGAnalyzeURL))
	}

	fmt.Fprintln(os.Stderr)
	if verbosity == VerbosityTerse {
		allOk := checkAllAspectsOk(status)
		if allOk {
			fmt.Fprintf(os.Stderr, "\t%s All features ok\n", GreenCheck)
			return
		}
	}

	monitoringConnIcon, monitoringConnMsg := getAspectStatus(status, state.CollectionAspectMonitoringDbConnection)
	fmt.Fprintf(os.Stderr, "\t%s Database connection:\t%s\n", monitoringConnIcon, monitoringConnMsg)

	pgVersionIcon, pgVersionMsg := getAspectStatus(status, state.CollectionAspectPgVersion)
	fmt.Fprintf(os.Stderr, "\t%s Postgres version:\t%s\n", pgVersionIcon, pgVersionMsg)

	pgssIcon, pgssMsg := getAspectStatus(status, state.CollectionAspectPgStatStatements)
	fmt.Fprintf(os.Stderr, "\t%s pg_stat_statements:\t%s\n", pgssIcon, pgssMsg)

	maxDbNameLen := getMaxDbNameLen(status.MonitoredDbs)
	schemaInfoIcon, schemaInfoSummaryMsg := summarizeDbChecks(status, state.CollectionAspectSchemaInformation, verbose)
	fmt.Fprintf(os.Stderr, "\t%s Schema information:\t%s\n", schemaInfoIcon, schemaInfoSummaryMsg)
	if verbose {
		for _, dbName := range status.MonitoredDbs {
			dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectSchemaInformation)

			printDbStatus(dbName, dbStatus, maxDbNameLen)
		}
	}
	colStatsIcon, colStatsSummaryMsg := summarizeDbChecks(status, state.CollectionAspectColumnStats, verbose)
	fmt.Fprintf(os.Stderr, "\t%s Column stats:\t\t%s\n", colStatsIcon, colStatsSummaryMsg)
	if verbose {
		for _, dbName := range status.MonitoredDbs {
			dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectColumnStats)

			printDbStatus(dbName, dbStatus, maxDbNameLen)
		}
	}
	extStatsIcon, extStatsSummaryMsg := summarizeDbChecks(status, state.CollectionAspectExtendedStats, verbose)
	fmt.Fprintf(os.Stderr, "\t%s Extended stats:\t%s\n", extStatsIcon, extStatsSummaryMsg)
	if verbose {
		for _, dbName := range status.MonitoredDbs {
			dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectExtendedStats)

			printDbStatus(dbName, dbStatus, maxDbNameLen)
		}
	}
	fmt.Fprintln(os.Stderr)

	qpIcon, qpMsg := getQueryPerformanceStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Query Performance:\t%s\n", qpIcon, qpMsg)

	iaIcon, iaMsg := getIndexAdvisorStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Index Advisor:\t%s\n", iaIcon, iaMsg)

	vaIcon, vaMsg := getVACUUMAdvisorStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s VACUUM Advisor:\t%s\n", vaIcon, vaMsg)

	explainIcon, explainMsg := getAutomatedExplainStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s EXPLAIN Plans:\t%s\n", explainIcon, explainMsg)

	ssIcon, ssMsg := getSchemaStatisticsStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Schema Statistics:\t%s\n", ssIcon, ssMsg)

	logsIcon, logMsg := getLogInsightsStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Log Insights:\t\t%s\n", logsIcon, logMsg)

	sysIcon, sysMsg := getAspectStatus(status, state.CollectionAspectSystemStats)
	fmt.Fprintf(os.Stderr, "\t%s System:\t\t%s\n", sysIcon, sysMsg)
}

func checkAllAspectsOk(status *state.SelfTestResult) bool {
	for _, aspect := range state.CollectionAspects {
		status := status.GetCollectionAspectStatus(aspect)
		if aspect == state.CollectionAspectExplain && (status == nil || status.State == state.CollectionStateUnchecked) {
			// We don't have a mechanism to verify this yet; so ignore it for the
			// purpose of this check
			continue
		}
		if status == nil || status.State != state.CollectionStateOkay {
			return false
		}
	}
	for _, aspect := range state.DbCollectionAspects {
		for _, dbName := range status.MonitoredDbs {
			status := status.GetDbCollectionAspectStatus(dbName, aspect)
			if status == nil || status.State != state.CollectionStateOkay {
				return false
			}
		}
	}
	return true
}

func getQueryPerformanceStatus(status *state.SelfTestResult) (string, string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required"
	}
	if s := status.GetCollectionAspectStatus(state.CollectionAspectPgStatStatements); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "pg_stat_statements required"
	}
	return GreenCheck, "ok"
}

func getSchemaStatisticsStatus(status *state.SelfTestResult) (string, string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required"
	}
	allDbsOkay := true
	someDbsOkay := false
	for _, dbName := range status.MonitoredDbs {
		dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectSchemaInformation)
		if dbStatus == nil || dbStatus.State != state.CollectionStateOkay {
			allDbsOkay = false
			break
		} else if !someDbsOkay {
			someDbsOkay = true
		}
	}
	if !someDbsOkay {
		return RedX, "not available due to errors; see above"
	}
	if !allDbsOkay {
		return YellowBang, "available for some databases"
	}
	return GreenCheck, "ok"
}

func getIndexAdvisorStatus(status *state.SelfTestResult) (string, string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required"
	}
	if len(status.MonitoredDbs) == 0 {
		return RedX, "could not check databases"
	}
	allDbsSchemaOk := true
	someDbsSchemaOk := false
	allDbsColStatsOk := true
	allDbsExtStatsOk := true
	for _, dbName := range status.MonitoredDbs {
		dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectSchemaInformation)
		if dbStatus == nil || dbStatus.State != state.CollectionStateOkay {
			allDbsSchemaOk = false
			break
		} else if !someDbsSchemaOk {
			someDbsSchemaOk = true
		}
		dbColStatsStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectColumnStats)
		if dbColStatsStatus == nil || dbColStatsStatus.State != state.CollectionStateOkay {
			allDbsColStatsOk = false
			break
		}
		allDbsExtStatsStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectExtendedStats)
		if allDbsExtStatsStatus == nil || allDbsExtStatsStatus.State != state.CollectionStateOkay {
			allDbsExtStatsOk = false
			break
		}

	}
	if !someDbsSchemaOk {
		return RedX, "not available due to schema monitoring errors; see above"
	}
	if !allDbsSchemaOk {
		return YellowBang, "schema monitoring errors in some databases; see above"
	}
	if !allDbsColStatsOk {
		return YellowBang, "column stats helper missing in some databases; see above"
	}
	if !allDbsExtStatsOk {
		return YellowBang, "extended stats helper missing in some databases; see above"
	}

	return GreenCheck, "ok"
}

func getVACUUMAdvisorStatus(status *state.SelfTestResult) (string, string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required"
	}

	if s := status.GetCollectionAspectStatus(state.CollectionAspectLogs); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "Log Insights required"
	}

	return GreenCheck, "ok"
}

func getLogInsightsStatus(status *state.SelfTestResult) (string, string) {
	return getAspectStatus(status, state.CollectionAspectLogs)
}

func getAutomatedExplainStatus(status *state.SelfTestResult) (string, string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required"
	}

	if s := status.GetCollectionAspectStatus(state.CollectionAspectLogs); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "Log Insights required"
	}

	// Right now, we don't have a test for this in the collector
	return GrayQuestion, "check pganalyze EXPLAIN Plans page"
}
