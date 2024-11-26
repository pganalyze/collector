package transform_test

import (
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStatements(t *testing.T) {
	key1 := state.PostgresStatementKey{QueryID: 1}
	key2 := state.PostgresStatementKey{QueryID: 2}
	pKey1 := state.PostgresPlanKey{PlanID: 111}
	pKey2 := state.PostgresPlanKey{PlanID: 222}
	pKey1.QueryID = key2.QueryID
	pKey2.QueryID = key2.QueryID

	newState := state.PersistedState{}
	transientState := state.TransientState{Statements: make(state.PostgresStatementMap), StatementTexts: make(state.PostgresStatementTextMap), Plans: make(state.PostgresPlanMap)}
	diffState := state.DiffState{StatementStats: make(state.DiffedPostgresStatementStatsMap), PlanStats: make(state.DiffedPostgresPlanStatsMap)}

	q1 := "SELECT 1"
	q2 := "SELECT * FROM test"
	fp1 := util.FingerprintQuery(q1, "none", -1)
	fpBuf1 := make([]byte, 8)
	binary.BigEndian.PutUint64(fpBuf1, fp1)
	fp2 := util.FingerprintQuery(q2, "none", -1)
	fpBuf2 := make([]byte, 8)
	binary.BigEndian.PutUint64(fpBuf2, fp2)
	capturedTime := time.Time{}
	transientState.Statements[key1] = state.PostgresStatement{Fingerprint: fp1}
	transientState.Statements[key2] = state.PostgresStatement{Fingerprint: fp2}
	transientState.StatementTexts[fp1] = q1
	transientState.StatementTexts[fp2] = q2
	transientState.Plans[pKey1] = state.PostgresPlan{ExplainPlan: "Index Scan", PlanCapturedTime: capturedTime}
	transientState.Plans[pKey2] = state.PostgresPlan{ExplainPlan: "Bitmap Heap Scan", PlanCapturedTime: capturedTime}
	diffState.StatementStats[key1] = state.DiffedPostgresStatementStats{Calls: 1}
	diffState.StatementStats[key2] = state.DiffedPostgresStatementStats{Calls: 13}
	diffState.PlanStats[pKey1] = state.DiffedPostgresStatementStats{Calls: 2}
	diffState.PlanStats[pKey2] = state.DiffedPostgresStatementStats{Calls: 24}

	actual := transform.StateToSnapshot(newState, diffState, transientState)
	actualJSON, _ := json.Marshal(actual)

	// Query: 0, 1, Plan: 0, 1 (w/ QueryIdx 1)
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
		ServerStatistic: &pganalyze_collector.ServerStatistic{},
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
		QueryPlanReferences: []*pganalyze_collector.QueryPlanReference{
			&pganalyze_collector.QueryPlanReference{
				QueryIdx:       1,
				OriginalPlanId: 111,
			},
			&pganalyze_collector.QueryPlanReference{
				QueryIdx:       1,
				OriginalPlanId: 222,
			},
		},
		QueryPlanInformations: []*pganalyze_collector.QueryPlanInformation{
			&pganalyze_collector.QueryPlanInformation{
				QueryPlanIdx:     0,
				ExplainPlan:      "Index Scan",
				PlanCapturedTime: timestamppb.New(capturedTime),
			},
			&pganalyze_collector.QueryPlanInformation{
				QueryPlanIdx:     1,
				ExplainPlan:      "Bitmap Heap Scan",
				PlanCapturedTime: timestamppb.New(capturedTime),
			},
		},
		QueryPlanStatistics: []*pganalyze_collector.QueryPlanStatistic{
			&pganalyze_collector.QueryPlanStatistic{
				QueryPlanIdx: 0,
				Calls:        2,
			},
			&pganalyze_collector.QueryPlanStatistic{
				QueryPlanIdx: 1,
				Calls:        24,
			},
		},
	}
	expectedJSON, _ := json.Marshal(expected)

	// Query: 1, 0, Plan: 0, 1 (w/ QueryIdx 0)
	var expectedAlt pganalyze_collector.FullSnapshot
	json.Unmarshal(expectedJSON, &expectedAlt)
	expectedAlt.QueryReferences = []*pganalyze_collector.QueryReference{
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
	}
	expectedAlt.QueryInformations = []*pganalyze_collector.QueryInformation{
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
	}
	expectedAlt.QueryStatistics = []*pganalyze_collector.QueryStatistic{
		&pganalyze_collector.QueryStatistic{
			QueryIdx: 0,
			Calls:    13,
		},
		&pganalyze_collector.QueryStatistic{
			QueryIdx: 1,
			Calls:    1,
		},
	}
	expectedAlt.QueryPlanReferences = []*pganalyze_collector.QueryPlanReference{
		&pganalyze_collector.QueryPlanReference{
			QueryIdx:       0,
			OriginalPlanId: 111,
		},
		&pganalyze_collector.QueryPlanReference{
			QueryIdx:       0,
			OriginalPlanId: 222,
		},
	}
	expectedJSONAlt, _ := json.Marshal(expectedAlt)

	// Query: 1, 0, Plan: 1, 0 (w/ QueryIdx 0)
	expectedAlt.QueryPlanReferences = []*pganalyze_collector.QueryPlanReference{
		&pganalyze_collector.QueryPlanReference{
			QueryIdx:       0,
			OriginalPlanId: 222,
		},
		&pganalyze_collector.QueryPlanReference{
			QueryIdx:       0,
			OriginalPlanId: 111,
		},
	}
	expectedAlt.QueryPlanInformations = []*pganalyze_collector.QueryPlanInformation{
		&pganalyze_collector.QueryPlanInformation{
			QueryPlanIdx:     0,
			ExplainPlan:      "Bitmap Heap Scan",
			PlanCapturedTime: timestamppb.New(capturedTime),
		},
		&pganalyze_collector.QueryPlanInformation{
			QueryPlanIdx:     1,
			ExplainPlan:      "Index Scan",
			PlanCapturedTime: timestamppb.New(capturedTime),
		},
	}
	expectedAlt.QueryPlanStatistics = []*pganalyze_collector.QueryPlanStatistic{
		&pganalyze_collector.QueryPlanStatistic{
			QueryPlanIdx: 0,
			Calls:        24,
		},
		&pganalyze_collector.QueryPlanStatistic{
			QueryPlanIdx: 1,
			Calls:        2,
		},
	}
	expectedJSONAlt2, _ := json.Marshal(expectedAlt)

	// Query: 0, 1, Plan: 1, 0 (w/ QueryIdx 1)
	expectedAlt.QueryReferences = []*pganalyze_collector.QueryReference{
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
	}
	expectedAlt.QueryInformations = []*pganalyze_collector.QueryInformation{
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
	}
	expectedAlt.QueryStatistics = []*pganalyze_collector.QueryStatistic{
		&pganalyze_collector.QueryStatistic{
			QueryIdx: 0,
			Calls:    1,
		},
		&pganalyze_collector.QueryStatistic{
			QueryIdx: 1,
			Calls:    13,
		},
	}
	expectedAlt.QueryPlanReferences = []*pganalyze_collector.QueryPlanReference{
		&pganalyze_collector.QueryPlanReference{
			QueryIdx:       1,
			OriginalPlanId: 222,
		},
		&pganalyze_collector.QueryPlanReference{
			QueryIdx:       1,
			OriginalPlanId: 111,
		},
	}
	expectedJSONAlt3, _ := json.Marshal(expectedAlt)

	if string(expectedJSON) != string(actualJSON) &&
		string(expectedJSONAlt) != string(actualJSON) &&
		string(expectedJSONAlt2) != string(actualJSON) &&
		string(expectedJSONAlt3) != string(actualJSON) {
		t.Errorf("\nExpected:%+v\n\tActual: %+v\n\n", string(expectedJSON), string(actualJSON))
	}
}
