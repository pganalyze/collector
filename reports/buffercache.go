package reports

import (
	"database/sql"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// BuffercacheReport - Report on the Postgres buffer cache
type BuffercacheReport struct {
	ReportRunID string
	CollectedAt time.Time
	Data        state.PostgresBuffercache
}

// RunID - Returns the ID of this report run
func (report BuffercacheReport) RunID() string {
	return report.ReportRunID
}

// ReportType - Returns the type of the report as a string
func (report BuffercacheReport) ReportType() string {
	return "buffercache"
}

// Run the report
func (report *BuffercacheReport) Run(server state.Server, logger *util.Logger, connection *sql.DB) (err error) {
	isAmazonRds := server.Config.SystemType == "amazon_rds"

	report.Data, err = postgres.GetBuffercache(logger, connection, isAmazonRds)
	if err != nil {
		return
	}

	return
}

// Result of the report
func (report *BuffercacheReport) Result() *pganalyze_collector.Report {
	var r pganalyze_collector.Report
	var data pganalyze_collector.BuffercacheReportData
	var exists bool

	r.ReportRunId = report.ReportRunID
	r.ReportType = "buffercache"
	r.CollectedAt, _ = ptypes.TimestampProto(report.CollectedAt)

	data.FreeBytes = report.Data.FreeBytes
	data.TotalBytes = report.Data.TotalBytes

	databaseNameToIdx := make(map[string]int32)

	for _, entry := range report.Data.Entries {
		e := pganalyze_collector.BuffercacheEntry{Bytes: entry.Bytes, Toast: entry.Toast}
		e.DatabaseIdx, exists = databaseNameToIdx[entry.DatabaseName]
		if !exists {
			ref := pganalyze_collector.DatabaseReference{Name: entry.DatabaseName}
			e.DatabaseIdx = int32(len(data.DatabaseReferences))
			data.DatabaseReferences = append(data.DatabaseReferences, &ref)
			databaseNameToIdx[entry.DatabaseName] = e.DatabaseIdx
		}
		if entry.SchemaName != nil {
			e.SchemaName = *entry.SchemaName
		}
		if entry.ObjectName != nil {
			e.ObjectName = *entry.ObjectName
		}
		if entry.ObjectKind != nil {
			e.ObjectKind = *entry.ObjectKind
		}
		data.BuffercacheEntries = append(data.BuffercacheEntries, &e)
	}

	r.Data = &pganalyze_collector.Report_BuffercacheReportData{BuffercacheReportData: &data}

	return &r
}
