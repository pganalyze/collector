package reports

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type Report interface {
	RunID() string
	ReportType() string
	Run(server state.Server, logger *util.Logger, connection *sql.DB) error
	Result() *pganalyze_collector.Report
}

var SupportedReports = []string{"bloat", "buffercache", "vacuum", "sequence"}

func InitializeReport(reportType string, reportRunID string) (Report, error) {
	switch reportType {
	case "bloat":
		return &BloatReport{ReportRunID: reportRunID, CollectedAt: time.Now()}, nil
	case "buffercache":
		return &BuffercacheReport{ReportRunID: reportRunID, CollectedAt: time.Now()}, nil
	case "vacuum":
		return &VacuumReport{ReportRunID: reportRunID, CollectedAt: time.Now()}, nil
	case "sequence":
		return &SequenceReport{ReportRunID: reportRunID, CollectedAt: time.Now()}, nil
	default:
		return nil, fmt.Errorf("Unknown report type: %s", reportType)
	}
}
