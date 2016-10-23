package reports

import (
	"github.com/golang/protobuf/proto"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// BuffercacheReport - Report on the Postgres buffer cache
type BuffercacheReport struct {
	Data state.PostgresBuffercache
}

// Run the report
func (report *BuffercacheReport) Run(server state.Server, logger *util.Logger) (err error) {
	report.Data, err = postgres.GetBuffercache(logger, server.Connection)
	if err != nil {
		return
	}

	return
}

// Result of the report
func (report *BuffercacheReport) Result() proto.Message {
	var r pganalyze_collector.BuffercacheReport
	var exists bool

	r.ReportRunUuid = "dummy"
	r.FreeBytes = report.Data.FreeBytes
	r.TotalBytes = report.Data.TotalBytes

	databaseNameToIdx := make(map[string]int32)

	for _, entry := range report.Data.Entries {
		e := pganalyze_collector.BuffercacheEntry{Bytes: entry.Bytes}
		e.DatabaseIdx, exists = databaseNameToIdx[entry.DatabaseName]
		if !exists {
			ref := pganalyze_collector.DatabaseReference{Name: entry.DatabaseName}
			e.DatabaseIdx = int32(len(r.DatabaseReferences))
			r.DatabaseReferences = append(r.DatabaseReferences, &ref)
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
		r.BuffercacheEntries = append(r.BuffercacheEntries, &e)
	}
	return &r
}
