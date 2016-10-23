package reports

import (
	"github.com/golang/protobuf/proto"
	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// BloatReport - Report on table and index bloat
type BloatReport struct {
	Data state.PostgresBloatStats
}

func (report *BloatReport) Run(server state.Server, logger *util.Logger) (err error) {
	report.Data, err = postgres.GetBloatStats(server.Connection)
	if err != nil {
		return
	}

	return
}

func (report *BloatReport) Result() proto.Message {
	var r pganalyze_collector.BloatReport

	r.ReportRunUuid = "dummy"
	r.DatabaseReferences = append(r.DatabaseReferences, &pganalyze_collector.DatabaseReference{Name: report.Data.DatabaseName})

	for _, relation := range report.Data.Relations {
		r.RelationBloatStatistics = append(r.RelationBloatStatistics, &pganalyze_collector.RelationBloatStatistic{RelationIdx: int32(len(r.RelationReferences)), BloatLookupMethod: pganalyze_collector.BloatLookupMethod_ESTIMATE_FAST, TotalBytes: relation.TotalBytes, BloatBytes: relation.BloatBytes})
		r.RelationReferences = append(r.RelationReferences, &pganalyze_collector.RelationReference{DatabaseIdx: 0, SchemaName: relation.SchemaName, RelationName: relation.RelationName})
	}

	for _, index := range report.Data.Indices {
		r.IndexBloatStatistics = append(r.IndexBloatStatistics, &pganalyze_collector.IndexBloatStatistic{IndexIdx: int32(len(r.IndexReferences)), TotalBytes: index.TotalBytes, BloatBytes: index.BloatBytes})
		r.IndexReferences = append(r.IndexReferences, &pganalyze_collector.IndexReference{DatabaseIdx: 0, SchemaName: index.SchemaName, IndexName: index.IndexName})
	}

	return &r
}
