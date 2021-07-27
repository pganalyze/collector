package output

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/pganalyze/collector/reports"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func submitReportRun(server *state.Server, report reports.Report, logger *util.Logger, s3Location string) error {
	data := url.Values{"s3_location": {s3Location}}

	req, err := http.NewRequest("POST", server.Config.APIBaseURL+"/v2/reports/submit_run", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("Pganalyze-System-Scope-Fallback", server.Config.SystemScopeFallback)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	resp, err := server.Config.HTTPClientWithRetry.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error when submitting: %s\n", body)
	}

	if len(body) > 0 {
		logger.PrintInfo("%s", body)
	} else {
		logger.PrintInfo("Submitted %s report successfully", report.ReportType())
	}

	return nil
}

func SubmitReport(server *state.Server, grant state.Grant, report reports.Report, logger *util.Logger) error {
	var err error
	var data []byte

	r := report.Result()

	data, err = proto.Marshal(r)
	if err != nil {
		logger.PrintError("Error marshaling protocol buffers")
		return err
	}

	var compressedData bytes.Buffer
	w := zlib.NewWriter(&compressedData)
	w.Write(data)
	w.Close()

	s3Location, err := uploadSnapshot(server.Config.HTTPClientWithRetry, grant, logger, compressedData, report.RunID())
	if err != nil {
		logger.PrintError("Error uploading to S3: %s", err)
		return err
	}

	return submitReportRun(server, report, logger, s3Location)
}
