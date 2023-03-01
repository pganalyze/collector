package reports

import (
	"context"
	"database/sql"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BloatReport - Report on table and index bloat
type BloatReport struct {
	ReportRunID string
	CollectedAt time.Time
	Data        state.PostgresBloatStats
}

// RunID - Returns the ID of this report run
func (report BloatReport) RunID() string {
	return report.ReportRunID
}

// ReportType - Returns the type of the report as a string
func (report BloatReport) ReportType() string {
	return "bloat"
}

// Run the report
func (report *BloatReport) Run(ctx context.Context, server *state.Server, logger *util.Logger, connection *sql.DB) (err error) {
	systemType := server.Config.SystemType

	report.Data, err = postgres.GetBloatStats(ctx, logger, connection, systemType, server.Config.IgnoreSchemaRegexp)
	if err != nil {
		return
	}

	return
}

// Result of the report
func (report *BloatReport) Result() *pganalyze_collector.Report {
	var r pganalyze_collector.Report
	var data pganalyze_collector.BloatReportData

	r.ReportRunId = report.ReportRunID
	r.ReportType = "bloat"
	r.CollectedAt = timestamppb.New(report.CollectedAt)

	data.DatabaseReferences = append(data.DatabaseReferences, &pganalyze_collector.DatabaseReference{Name: report.Data.DatabaseName})

	for _, relation := range report.Data.Relations {
		data.RelationBloatStatistics = append(data.RelationBloatStatistics, &pganalyze_collector.RelationBloatStatistic{RelationIdx: int32(len(data.RelationReferences)), BloatLookupMethod: pganalyze_collector.BloatLookupMethod_ESTIMATE_FAST, TotalBytes: relation.TotalBytes, BloatBytes: relation.BloatBytes})
		data.RelationReferences = append(data.RelationReferences, &pganalyze_collector.RelationReference{DatabaseIdx: 0, SchemaName: relation.SchemaName, RelationName: relation.RelationName})
	}

	for _, index := range report.Data.Indices {
		data.IndexBloatStatistics = append(data.IndexBloatStatistics, &pganalyze_collector.IndexBloatStatistic{IndexIdx: int32(len(data.IndexReferences)), TotalBytes: index.TotalBytes, BloatBytes: index.BloatBytes})
		data.IndexReferences = append(data.IndexReferences, &pganalyze_collector.IndexReference{DatabaseIdx: 0, SchemaName: index.SchemaName, IndexName: index.IndexName})
	}

	r.Data = &pganalyze_collector.Report_BloatReportData{BloatReportData: &data}

	return &r
}
