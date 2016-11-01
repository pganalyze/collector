package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/protobuf/jsonpb"

	"github.com/pganalyze/collector/output"
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

	report, err = reports.InitializeReport(reportType, "dummy")
	if err != nil {
		logger.PrintError("Failed to initialize report: %s", err)
		return nil
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

type RequestedReport struct {
	ReportType  string `json:"report_type"`
	ReportRunID string `json:"report_run_id"`
}

type reportsApiResponse struct {
	RequestedReports []RequestedReport `json:"requested_reports"`
	Grant            *state.Grant
}

func getRequestedReports(server state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (requestedReports []reports.Report, grant state.Grant, err error) {
	data := url.Values{"supported_reports": {strings.Join(reports.SupportedReports, ",")}}
	req, err := http.NewRequest("POST", server.Config.APIBaseURL+"/v2/reports/fetch_runs", strings.NewReader(data.Encode()))
	if err != nil {
		return
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		err = fmt.Errorf("Error when getting requested reports: %s\n", body)
		return
	}

	parsedBody := reportsApiResponse{}
	err = json.Unmarshal(body, &parsedBody)
	if err != nil {
		return
	}

	for _, r := range parsedBody.RequestedReports {
		requestedReport, err := reports.InitializeReport(r.ReportType, r.ReportRunID)
		if err != nil {
			logger.PrintWarning("Ignoring report request due to error: %s", err)
			// TODO: This should also tell the server we encountered an error with this report run
			continue
		}
		requestedReports = append(requestedReports, requestedReport)
	}

	if parsedBody.Grant != nil {
		grant = *parsedBody.Grant
		grant.Valid = true
	}

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
	var err error

	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)

		server.Connection, err = establishConnection(server, prefixedLogger, globalCollectionOpts)
		if err != nil {
			prefixedLogger.PrintError("Error: Failed to connect to database: %s", err)
			continue
		}

		reports, grant, err := getRequestedReports(server, globalCollectionOpts, prefixedLogger)
		if err != nil {
			prefixedLogger.PrintError("Failed to get requested reports: %s", err)
			continue
		}

		for _, report := range reports {
			err = report.Run(server, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("Failed to run report: %s", err)
				return
			}

			output.SubmitReport(server, grant, report, prefixedLogger)
		}

		// This is the easiest way to avoid opening multiple connections to different databases on the same instance
		server.Connection.Close()
		server.Connection = nil
	}
}
