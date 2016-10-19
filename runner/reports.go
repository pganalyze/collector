package runner

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/jsonpb"

	"github.com/pganalyze/collector/reports"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func runReport(reportType string, server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (report reports.Report) {
	var err error

	prefixedLogger := logger.WithPrefix(server.Config.SectionName)

	server.Connection, err = establishConnection(server, logger, globalCollectionOpts)
	if err != nil {
		prefixedLogger.PrintError("Error: Failed to connect to database: %s", err)
		return
	}

	switch reportType {
	case "bloat":
		report = &reports.BloatReport{}
	case "buffercache":
		report = &reports.BuffercacheReport{}
	default:
		panic("YO?")
		// If test run, err out with unknown report type
	}

	err = report.Run(server, logger)
	if err != nil {
		logger.PrintError("Failed to run report: %s", err)
		return nil
	}

	// This is the easiest way to avoid opening multiple connections to different databases on the same instance
	server.Connection.Close()
	server.Connection = nil

	return
}

// RunTestReport - Runs globalCollectionOpts.TestReport for all servers and outputs the result to stdout
func RunTestReport(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	for _, server := range servers {
		report := runReport(globalCollectionOpts.TestReport, server, globalCollectionOpts, logger)
		if report == nil {
			continue
		}

		var out bytes.Buffer
		var marshaler jsonpb.Marshaler
		dataJSON, err := marshaler.MarshalToString(report.Result())
		if err != nil {
			logger.PrintError("Failed to transform protocol buffers to JSON: %s", err)
			return
		}
		json.Indent(&out, []byte(dataJSON), "", "\t")
		fmt.Printf("%s\n", out.String())
	}
}

// RunRequestedReports - Retrieves current report requests from the server, runs them and submits their data
func RunRequestedReports(servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	var reports []reports.Report
	for _, server := range servers {
		report := runReport(globalCollectionOpts.TestReport, server, globalCollectionOpts, logger)
		if report == nil {
			continue
		}
		reports = append(reports, report)
	}
	// Needs to call output/reports.go with the results
}
