package reports

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// VacuumReport - Report on sequence statistics
type SequenceReport struct {
	ReportRunID string
	CollectedAt time.Time
	Data        state.PostgresSequenceReport
}

// RunID - Returns the ID of this report run
func (report SequenceReport) RunID() string {
	return report.ReportRunID
}

// ReportType - Returns the type of the report as a string
func (report SequenceReport) ReportType() string {
	return "sequence"
}

// Run the report
func (report *SequenceReport) Run(server state.Server, logger *util.Logger) (err error) {
	report.Data, err = postgres.GetSequenceReport(logger, server.Connection)
	if err != nil {
		return
	}

	return
}

// Result of the report
func (report *SequenceReport) Result() *pganalyze_collector.Report {
	var r pganalyze_collector.Report
	var data pganalyze_collector.SequenceReportData

	r.ReportRunId = report.ReportRunID
	r.ReportType = report.ReportType()
	r.CollectedAt, _ = ptypes.TimestampProto(report.CollectedAt)

	data.DatabaseReferences = append(data.DatabaseReferences, &pganalyze_collector.DatabaseReference{Name: report.Data.DatabaseName})

	sequenceOidToIdx := make(map[state.Oid]int32)

	for oid, s := range report.Data.Sequences {
		sequenceOidToIdx[oid] = int32(len(data.SequenceReferences))
		sInfo := pganalyze_collector.SequenceInformation{
			SequenceIdx: int32(len(data.SequenceReferences)),
			LastValue:   s.LastValue,
			StartValue:  s.StartValue,
			IncrementBy: s.IncrementBy,
			MaxValue:    s.MaxValue,
			MinValue:    s.MinValue,
			CacheValue:  s.CacheValue,
			IsCycled:    s.IsCycled,
		}
		data.SequenceInformations = append(data.SequenceInformations, &sInfo)
		data.SequenceReferences = append(data.SequenceReferences, &pganalyze_collector.SequenceReference{
			DatabaseIdx:  0,
			SchemaName:   s.SchemaName,
			SequenceName: s.SequenceName,
		})
	}

	relationOidToIdx := make(map[state.Oid]int32)

	for _, c := range report.Data.SerialColumns {
		relationOidToIdx[c.RelationOid] = int32(len(data.RelationReferences))
		data.RelationReferences = append(data.RelationReferences, &pganalyze_collector.RelationReference{
			DatabaseIdx:  0,
			SchemaName:   c.SchemaName,
			RelationName: c.RelationName,
		})

		cInfo := pganalyze_collector.SerialColumnInformation{
			RelationIdx:  relationOidToIdx[c.RelationOid],
			ColumnName:   c.ColumnName,
			DataType:     c.DataType,
			MaximumValue: c.MaximumValue,
			SequenceIdx:  sequenceOidToIdx[c.SequenceOid],
		}

		for _, fc := range c.ForeignColumns {
			relIdx, exists := relationOidToIdx[fc.RelationOid]
			if !exists {
				relIdx = int32(len(data.RelationReferences))
				relationOidToIdx[fc.RelationOid] = relIdx
				data.RelationReferences = append(data.RelationReferences, &pganalyze_collector.RelationReference{
					DatabaseIdx:  0,
					SchemaName:   fc.SchemaName,
					RelationName: fc.RelationName,
				})
			}
			fcInfo := pganalyze_collector.SerialColumnInformation_ForeignColumn{
				RelationIdx:  relIdx,
				ColumnName:   fc.ColumnName,
				DataType:     fc.DataType,
				MaximumValue: fc.MaximumValue,
				Inferred:     fc.Inferred,
			}
			cInfo.ForeignColumns = append(cInfo.ForeignColumns, &fcInfo)
		}

		data.SerialColumnInformations = append(data.SerialColumnInformations, &cInfo)
	}

	r.Data = &pganalyze_collector.Report_SequenceReportData{SequenceReportData: &data}

	return &r
}
