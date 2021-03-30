package transform_test

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStatements(t *testing.T) {
	key1 := state.PostgresStatementKey{QueryID: 1}
	key2 := state.PostgresStatementKey{QueryID: 2}

	newState := state.PersistedState{}
	transientState := state.TransientState{Statements: make(state.PostgresStatementMap), StatementTexts: make(state.PostgresStatementTextMap)}
	diffState := state.DiffState{StatementStats: make(state.DiffedPostgresStatementStatsMap)}

	q1 := "SELECT 1"
	q2 := "SELECT * FROM test"
	fp1 := util.FingerprintQuery(q1)
	fpBuf1 := make([]byte, 8)
	binary.BigEndian.PutUint64(fpBuf1, fp1)
	fp2 := util.FingerprintQuery(q2)
	fpBuf2 := make([]byte, 8)
	binary.BigEndian.PutUint64(fpBuf2, fp2)
	transientState.Statements[key1] = state.PostgresStatement{Fingerprint: fp1}
	transientState.Statements[key2] = state.PostgresStatement{Fingerprint: fp2}
	transientState.StatementTexts[fp1] = q1
	transientState.StatementTexts[fp2] = q2
	diffState.StatementStats[key1] = state.DiffedPostgresStatementStats{Calls: 1}
	diffState.StatementStats[key2] = state.DiffedPostgresStatementStats{Calls: 13}

	actual := transform.StateToSnapshot(newState, diffState, transientState)
	actualJSON, _ := json.Marshal(actual)

	expected := pganalyze_collector.FullSnapshot{
		Config:             &pganalyze_collector.CollectorConfig{},
		CollectorStatistic: &pganalyze_collector.CollectorStatistic{},
		CollectorStartedAt: &timestamppb.Timestamp{
			Seconds: -62135596800,
			Nanos:   0,
		},
		System: &pganalyze_collector.System{
			SystemInformation:  &pganalyze_collector.SystemInformation{},
			SchedulerStatistic: &pganalyze_collector.SchedulerStatistic{},
			MemoryStatistic:    &pganalyze_collector.MemoryStatistic{},
			CpuInformation:     &pganalyze_collector.CPUInformation{},
		},
		PostgresVersion: &pganalyze_collector.PostgresVersion{},
		Replication:     &pganalyze_collector.Replication{},
		QueryReferences: []*pganalyze_collector.QueryReference{
			&pganalyze_collector.QueryReference{
				DatabaseIdx: 0,
				RoleIdx:     0,
				Fingerprint: fpBuf1,
			},
			&pganalyze_collector.QueryReference{
				DatabaseIdx: 0,
				RoleIdx:     0,
				Fingerprint: fpBuf2,
			},
		},
		QueryInformations: []*pganalyze_collector.QueryInformation{
			&pganalyze_collector.QueryInformation{
				QueryIdx:        0,
				NormalizedQuery: "SELECT 1",
				QueryIds:        []int64{1},
			},
			&pganalyze_collector.QueryInformation{
				QueryIdx:        1,
				NormalizedQuery: "SELECT * FROM test",
				QueryIds:        []int64{2},
			},
		},
		QueryStatistics: []*pganalyze_collector.QueryStatistic{
			&pganalyze_collector.QueryStatistic{
				QueryIdx: 0,
				Calls:    1,
			},
			&pganalyze_collector.QueryStatistic{
				QueryIdx: 1,
				Calls:    13,
			},
		},
	}
	expectedJSON, _ := json.Marshal(expected)

	// Sadly this is the quickest way with all the idx references...
	expectedAlt := pganalyze_collector.FullSnapshot{
		Config:             &pganalyze_collector.CollectorConfig{},
		CollectorStatistic: &pganalyze_collector.CollectorStatistic{},
		CollectorStartedAt: &timestamppb.Timestamp{
			Seconds: -62135596800,
			Nanos:   0,
		},
		System: &pganalyze_collector.System{
			SystemInformation:  &pganalyze_collector.SystemInformation{},
			SchedulerStatistic: &pganalyze_collector.SchedulerStatistic{},
			MemoryStatistic:    &pganalyze_collector.MemoryStatistic{},
			CpuInformation:     &pganalyze_collector.CPUInformation{},
		},
		PostgresVersion: &pganalyze_collector.PostgresVersion{},
		Replication:     &pganalyze_collector.Replication{},
		QueryReferences: []*pganalyze_collector.QueryReference{
			&pganalyze_collector.QueryReference{
				DatabaseIdx: 0,
				RoleIdx:     0,
				Fingerprint: fpBuf2,
			},
			&pganalyze_collector.QueryReference{
				DatabaseIdx: 0,
				RoleIdx:     0,
				Fingerprint: fpBuf1,
			},
		},
		QueryInformations: []*pganalyze_collector.QueryInformation{
			&pganalyze_collector.QueryInformation{
				QueryIdx:        0,
				NormalizedQuery: "SELECT * FROM test",
				QueryIds:        []int64{2},
			},
			&pganalyze_collector.QueryInformation{
				QueryIdx:        1,
				NormalizedQuery: "SELECT 1",
				QueryIds:        []int64{1},
			},
		},
		QueryStatistics: []*pganalyze_collector.QueryStatistic{
			&pganalyze_collector.QueryStatistic{
				QueryIdx: 0,
				Calls:    13,
			},
			&pganalyze_collector.QueryStatistic{
				QueryIdx: 1,
				Calls:    1,
			},
		},
	}
	expectedJSONAlt, _ := json.Marshal(expectedAlt)

	if string(expectedJSON) != string(actualJSON) && string(expectedJSONAlt) != string(actualJSON) {
		t.Errorf("\nExpected:%+v\n\tActual: %+v\n\n", string(expectedJSON), string(actualJSON))
	}
}
