package reports

import (
	"database/sql"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// VacuumReport - Report on table vacuum statistics
type VacuumReport struct {
	ReportRunID string
	CollectedAt time.Time
	Data        state.PostgresVacuumStats
}

// RunID - Returns the ID of this report run
func (report VacuumReport) RunID() string {
	return report.ReportRunID
}

// ReportType - Returns the type of the report as a string
func (report VacuumReport) ReportType() string {
	return "vacuum"
}

// Run the report
func (report *VacuumReport) Run(server *state.Server, logger *util.Logger, connection *sql.DB) (err error) {
	report.Data, err = postgres.GetVacuumStats(logger, connection, server.Config.IgnoreSchemaRegexp)
	if err != nil {
		return
	}

	return
}

// Result of the report
func (report *VacuumReport) Result() *pganalyze_collector.Report {
	var r pganalyze_collector.Report
	var data pganalyze_collector.VacuumReportData

	r.ReportRunId = report.ReportRunID
	r.ReportType = report.ReportType()
	r.CollectedAt = timestamppb.New(report.CollectedAt)

	data.DatabaseReferences = append(data.DatabaseReferences, &pganalyze_collector.DatabaseReference{Name: report.Data.DatabaseName})
	data.AutovacuumMaxWorkers = report.Data.AutovacuumMaxWorkers
	data.AutovacuumNaptimeSeconds = report.Data.AutovacuumNaptimeSeconds

	for _, relation := range report.Data.Relations {
		stats := pganalyze_collector.VacuumStatistic{
			RelationIdx:                     int32(len(data.RelationReferences)),
			LiveRowCount:                    relation.LiveRowCount,
			DeadRowCount:                    relation.DeadRowCount,
			Relfrozenxid:                    relation.Relfrozenxid,
			Relminmxid:                      relation.Relminmxid,
			AutovacuumEnabled:               relation.AutovacuumEnabled,
			AutovacuumVacuumThreshold:       relation.AutovacuumVacuumThreshold,
			AutovacuumAnalyzeThreshold:      relation.AutovacuumAnalyzeThreshold,
			AutovacuumVacuumScaleFactor:     relation.AutovacuumVacuumScaleFactor,
			AutovacuumAnalyzeScaleFactor:    relation.AutovacuumAnalyzeScaleFactor,
			AutovacuumFreezeMaxAge:          relation.AutovacuumFreezeMaxAge,
			AutovacuumMultixactFreezeMaxAge: relation.AutovacuumMultixactFreezeMaxAge,
			AutovacuumVacuumCostDelay:       relation.AutovacuumVacuumCostDelay,
			AutovacuumVacuumCostLimit:       relation.AutovacuumVacuumCostLimit,
			Fillfactor:                      relation.Fillfactor,
		}

		if relation.LastManualVacuumRun.Valid {
			t := timestamppb.New(relation.LastManualVacuumRun.Time)
			stats.LastManualVacuumRun = &pganalyze_collector.NullTimestamp{Valid: true, Value: t}
		}
		if relation.LastAutoVacuumRun.Valid {
			t := timestamppb.New(relation.LastAutoVacuumRun.Time)
			stats.LastAutoVacuumRun = &pganalyze_collector.NullTimestamp{Valid: true, Value: t}
		}
		if relation.LastManualAnalyzeRun.Valid {
			t := timestamppb.New(relation.LastManualAnalyzeRun.Time)
			stats.LastManualAnalyzeRun = &pganalyze_collector.NullTimestamp{Valid: true, Value: t}
		}
		if relation.LastAutoAnalyzeRun.Valid {
			t := timestamppb.New(relation.LastAutoAnalyzeRun.Time)
			stats.LastAutoAnalyzeRun = &pganalyze_collector.NullTimestamp{Valid: true, Value: t}
		}

		data.VacuumStatistics = append(data.VacuumStatistics, &stats)
		data.RelationReferences = append(data.RelationReferences, &pganalyze_collector.RelationReference{
			DatabaseIdx:  0,
			SchemaName:   relation.SchemaName,
			RelationName: relation.RelationName,
		})
	}

	r.Data = &pganalyze_collector.Report_VacuumReportData{VacuumReportData: &data}

	return &r
}
