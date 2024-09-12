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
	var allChecked = true
	for _, dbName := range dbNames {
		if item, ok := checks[dbName]; ok && item.State == state.CollectionStateUnchecked {
			allChecked = false
			firstUncheckedDb = dbName
			break
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
	if len(dbNames) == 0 || len(checks) == 0 {
		summaryMsg = "could not check databases"
	} else if !allChecked {
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

func summarizeDbHints(status *state.SelfTestResult, aspect state.DbCollectionAspect) []string {
	hintsSet := make(map[string]bool)
	dbAspectStatuses := status.AllDbAspectStatuses[aspect]
	if len(dbAspectStatuses) == 0 {
		return nil
	}
	for _, status := range dbAspectStatuses {
		hintsSet[status.Hint] = true
	}
	var allHints []string
	for hint := range hintsSet {
		allHints = append(allHints, hint)
	}
	return allHints
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

func getAspectStatus(status *state.SelfTestResult, aspect state.CollectionAspect) (icon string, msg string, hint string) {
	aspectStatus := status.GetCollectionAspectStatus(aspect)
	if aspectStatus == nil {
		return getStatusIcon(state.CollectionStateUnchecked), "", ""
	}
	return getStatusIcon(aspectStatus.State), aspectStatus.Msg, aspectStatus.Hint
}

func printAspectStatus(status *state.SelfTestResult, aspect state.CollectionAspect, label string) {
	const maxLabelLen = len("pganalyze connection")
	icon, msg, hint := getAspectStatus(status, aspect)

	// Ensure that status labels names of different lengths line up
	labelFmtString := fmt.Sprintf("%%%ds", maxLabelLen-len(label))
	fmt.Fprintf(os.Stderr, "\t%s %s:"+labelFmtString+"\t%s\n", icon, label, "", msg)
	printHint(hint)
}

func printHint(hint string) {
	if hint != "" {
		fmt.Fprintf(os.Stderr, "\t\tHINT:\t%s\n", hint)
	}
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

	printAspectStatus(status, state.CollectionAspectApiConnection, "pganalyze connection")
	printAspectStatus(status, state.CollectionAspectWebSocket, "pganalyze WebSocket")
	printAspectStatus(status, state.CollectionAspectTelemetry, "Collector telemetry")

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

	printAspectStatus(status, state.CollectionAspectMonitoringDbConnection, "Database connection")
	printAspectStatus(status, state.CollectionAspectPgVersion, "Postgres version")
	printAspectStatus(status, state.CollectionAspectPgStatStatements, "pg_stat_statements")

	maxDbNameLen := getMaxDbNameLen(status.MonitoredDbs)
	schemaIcon, schemaSummaryMsg := summarizeDbChecks(status, state.CollectionAspectSchema, verbose)
	fmt.Fprintf(os.Stderr, "\t%s Schema information:\t%s\n", schemaIcon, schemaSummaryMsg)
	if verbose {
		for _, dbName := range status.MonitoredDbs {
			dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectSchema)

			printDbStatus(dbName, dbStatus, maxDbNameLen)
		}
	}
	schemaHints := summarizeDbHints(status, state.CollectionAspectSchema)
	for _, hint := range schemaHints {
		printHint(hint)
	}

	colStatsIcon, colStatsSummaryMsg := summarizeDbChecks(status, state.CollectionAspectColumnStats, verbose)
	fmt.Fprintf(os.Stderr, "\t%s Column stats:\t\t%s\n", colStatsIcon, colStatsSummaryMsg)
	if verbose {
		for _, dbName := range status.MonitoredDbs {
			dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectColumnStats)

			printDbStatus(dbName, dbStatus, maxDbNameLen)
		}
	}
	colStatsHints := summarizeDbHints(status, state.CollectionAspectColumnStats)
	for _, hint := range colStatsHints {
		printHint(hint)
	}

	extStatsIcon, extStatsSummaryMsg := summarizeDbChecks(status, state.CollectionAspectExtendedStats, verbose)
	fmt.Fprintf(os.Stderr, "\t%s Extended stats:\t%s\n", extStatsIcon, extStatsSummaryMsg)
	if verbose {
		for _, dbName := range status.MonitoredDbs {
			dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectExtendedStats)

			printDbStatus(dbName, dbStatus, maxDbNameLen)
		}
	}
	extStatsHints := summarizeDbHints(status, state.CollectionAspectExtendedStats)
	for _, hint := range extStatsHints {
		printHint(hint)
	}
	fmt.Fprintln(os.Stderr)

	qpIcon, qpMsg, qpHint := getQueryPerformanceStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Query Performance:\t%s\n", qpIcon, qpMsg)
	printHint(qpHint)

	iaIcon, iaMsg, iaHint := getIndexAdvisorStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Index Advisor:\t%s\n", iaIcon, iaMsg)
	printHint(iaHint)

	vaIcon, vaMsg, vaHint := getVACUUMAdvisorStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s VACUUM Advisor:\t%s\n", vaIcon, vaMsg)
	printHint(vaHint)

	explainIcon, explainMsg, explainHint := getAutomatedExplainStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s EXPLAIN Plans:\t%s\n", explainIcon, explainMsg)
	printHint(explainHint)

	ssIcon, ssMsg, ssHint := getSchemaStatisticsStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Schema Statistics:\t%s\n", ssIcon, ssMsg)
	printHint(ssHint)

	logsIcon, logsMsg, logsHint := getLogInsightsStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Log Insights:\t\t%s\n", logsIcon, logsMsg)
	printHint(logsHint)

	connsIcon, connsMsg, connsHint := getConnectionsStatus(status)
	fmt.Fprintf(os.Stderr, "\t%s Connections:\t\t%s\n", connsIcon, connsMsg)
	printHint(connsHint)

	printAspectStatus(status, state.CollectionAspectSystemStats, "System")
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

func getQueryPerformanceStatus(status *state.SelfTestResult) (icon string, msg string, hint string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required", ""
	}
	if s := status.GetCollectionAspectStatus(state.CollectionAspectPgStatStatements); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "pg_stat_statements required", ""
	}
	return GreenCheck, "ok", ""
}

func getSchemaStatisticsStatus(status *state.SelfTestResult) (icon string, msg string, hint string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required", ""
	}
	allDbsOkay := true
	someDbsOkay := false
	for _, dbName := range status.MonitoredDbs {
		dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectSchema)
		if dbStatus == nil || dbStatus.State != state.CollectionStateOkay {
			allDbsOkay = false
			break
		} else if !someDbsOkay {
			someDbsOkay = true
		}
	}
	if !someDbsOkay {
		return RedX, "not available due to errors; see above", ""
	}
	if !allDbsOkay {
		return YellowBang, "available for some databases", ""
	}
	return GreenCheck, "ok", ""
}

func getIndexAdvisorStatus(status *state.SelfTestResult) (string, string, string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required", ""
	}
	if len(status.MonitoredDbs) == 0 {
		return RedX, "could not check databases", ""
	}
	allDbsSchemaOk := true
	someDbsSchemaOk := false
	allDbsColStatsOk := true
	allDbsExtStatsOk := true
	for _, dbName := range status.MonitoredDbs {
		dbStatus := status.GetDbCollectionAspectStatus(dbName, state.CollectionAspectSchema)
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
		return RedX, "not available due to schema monitoring errors; see above", ""
	}
	if !allDbsSchemaOk {
		return YellowBang, "schema monitoring errors in some databases; see above", "Schema information is required for Index Advisor"
	}
	if !allDbsColStatsOk {
		return YellowBang, "column stats helper missing in some databases; see above", "Column stats can improve index recommendations"
	}
	if !allDbsExtStatsOk {
		return YellowBang, "extended stats helper missing in some databases; see above", "Extended stats can improve index recommendations"
	}

	return GreenCheck, "ok", ""
}

func getVACUUMAdvisorStatus(status *state.SelfTestResult) (string, string, string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required", ""
	}

	// See note in getLogInsightsStatus
	if s := status.GetCollectionAspectStatus(state.CollectionAspectActivity); s != nil && s.State == state.CollectionStateNotAvailable {
		return getAspectStatus(status, state.CollectionAspectActivity)
	}

	if s := status.GetCollectionAspectStatus(state.CollectionAspectLogs); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "Log Insights required", ""
	}

	return GreenCheck, "ok", ""
}

func getLogInsightsStatus(status *state.SelfTestResult) (string, string, string) {
	// N.B.: We check the activity status here, because we don't check the Logs
	// grant on most platform, but we know if activity snapshots are not
	// available, log snapshots are not available either based on our current
	// plans
	if s := status.GetCollectionAspectStatus(state.CollectionAspectActivity); s != nil && s.State == state.CollectionStateNotAvailable {
		return getAspectStatus(status, state.CollectionAspectActivity)
	}

	return getAspectStatus(status, state.CollectionAspectLogs)
}

func getConnectionsStatus(status *state.SelfTestResult) (string, string, string) {
	return getAspectStatus(status, state.CollectionAspectActivity)
}

func getAutomatedExplainStatus(status *state.SelfTestResult) (string, string, string) {
	if s := status.GetCollectionAspectStatus(state.CollectionAspectMonitoringDbConnection); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "database connection required", ""
	}

	// See note in getLogInsightsStatus
	if s := status.GetCollectionAspectStatus(state.CollectionAspectActivity); s != nil && s.State == state.CollectionStateNotAvailable {
		return getAspectStatus(status, state.CollectionAspectActivity)
	}

	if s := status.GetCollectionAspectStatus(state.CollectionAspectLogs); s == nil || s.State != state.CollectionStateOkay {
		return RedX, "Log Insights required", ""
	}

	// Right now, we don't have a test for this in the collector
	return GrayQuestion, "check pganalyze EXPLAIN Plans page", ""
}
