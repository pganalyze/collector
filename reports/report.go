package reports

import (
	"fmt"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type Report interface {
	RunID() string
	ReportType() string
	Run(server state.Server, logger *util.Logger) error
	Result() *pganalyze_collector.Report
}

var SupportedReports = []string{"bloat", "buffercache"}

func InitializeReport(reportType string, reportRunID string) (Report, error) {
	switch reportType {
	case "bloat":
		return &BloatReport{ReportRunID: reportRunID, CollectedAt: time.Now()}, nil
	case "buffercache":
		return &BuffercacheReport{ReportRunID: reportRunID, CollectedAt: time.Now()}, nil
	default:
		return nil, fmt.Errorf("Unknown report type: %s", reportType)
	}
}
