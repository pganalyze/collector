package transform_test

import (
	"encoding/binary"
	"encoding/json"
	"slices"
	"sort"
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
	makeCanonical(actual)
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
		ServerStatistic: &pganalyze_collector.ServerStatistic{},
		Replication:     &pganalyze_collector.Replication{},
		QueryReferences: []*pganalyze_collector.QueryReference{
			{
				DatabaseIdx: 0,
				RoleIdx:     0,
				Fingerprint: fpBuf1,
			},
			{
				DatabaseIdx: 0,
				RoleIdx:     0,
				Fingerprint: fpBuf2,
			},
		},
		QueryInformations: []*pganalyze_collector.QueryInformation{
			{
				QueryIdx:        0,
				NormalizedQuery: "SELECT 1",
				QueryIds:        []int64{1},
			},
			{
				QueryIdx:        1,
				NormalizedQuery: "SELECT * FROM test",
				QueryIds:        []int64{2},
			},
		},
		QueryStatistics: []*pganalyze_collector.QueryStatistic{
			{
				QueryIdx: 0,
				Calls:    1,
			},
			{
				QueryIdx: 1,
				Calls:    13,
			},
		},
		QueryPlanReferences: []*pganalyze_collector.QueryPlanReference{
			{
				QueryIdx:       1,
				OriginalPlanId: 111,
			},
			{
				QueryIdx:       1,
				OriginalPlanId: 222,
			},
		},
		QueryPlanInformations: []*pganalyze_collector.QueryPlanInformation{
			{
				QueryPlanIdx:     0,
				ExplainPlan:      "Index Scan",
				PlanCapturedTime: timestamppb.New(capturedTime),
			},
			{
				QueryPlanIdx:     1,
				ExplainPlan:      "Bitmap Heap Scan",
				PlanCapturedTime: timestamppb.New(capturedTime),
			},
		},
		QueryPlanStatistics: []*pganalyze_collector.QueryPlanStatistic{
			{
				QueryPlanIdx: 0,
				Calls:        2,
			},
			{
				QueryPlanIdx: 1,
				Calls:        24,
			},
		},
	}
	makeCanonical(expected)
	expectedJSON, _ := json.Marshal(expected)

	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("\nExpected:%+v\n\tActual: %+v\n\n", string(expectedJSON), string(actualJSON))
	}
}

type OriginalQueryRef struct {
	Original    *pganalyze_collector.QueryReference
	OriginalIdx int32
}

type OriginalPlanRef struct {
	Original    *pganalyze_collector.QueryPlanReference
	OriginalIdx int32
}

// Takes a snapshot and rewrites it to a canonical form, with QueryReferences
// and similar fields sorted into a single canonical order.
func makeCanonical(snapshot pganalyze_collector.FullSnapshot) {
	// ensure query references occur in a consistent order
	queryRefs := make([]OriginalQueryRef, len(snapshot.QueryReferences))
	for i, qRef := range snapshot.QueryReferences {
		queryRefs[i] = OriginalQueryRef{
			Original:    qRef,
			OriginalIdx: int32(i),
		}
	}

	sort.Slice(queryRefs, func(i, j int) bool {
		a := queryRefs[i].Original
		b := queryRefs[j].Original

		if a.DatabaseIdx < b.DatabaseIdx {
			return true
		}
		if a.DatabaseIdx > b.DatabaseIdx {
			return false
		}
		if a.RoleIdx < b.RoleIdx {
			return true
		}
		if a.RoleIdx > b.RoleIdx {
			return true
		}
		if len(a.Fingerprint) < len(b.Fingerprint) {
			return true
		}
		if len(a.Fingerprint) > len(b.Fingerprint) {
			return false
		}
		for i, aFpByte := range a.Fingerprint {
			bFpByte := b.Fingerprint[i]
			if aFpByte < bFpByte {
				return true
			}
			if aFpByte > bFpByte {
				return false
			}
		}
		return false
	})

	for i, qRef := range queryRefs {
		snapshot.QueryReferences[i] = qRef.Original

		qInfoIdx := slices.IndexFunc(snapshot.QueryInformations, func(item *pganalyze_collector.QueryInformation) bool {
			return item.QueryIdx == qRef.OriginalIdx
		})
		qInfo := snapshot.QueryInformations[qInfoIdx]
		qInfo.QueryIdx = int32(i)

		qStatsIdx := slices.IndexFunc(snapshot.QueryStatistics, func(item *pganalyze_collector.QueryStatistic) bool {
			return item.QueryIdx == qRef.OriginalIdx
		})
		qStats := snapshot.QueryStatistics[qStatsIdx]
		qStats.QueryIdx = int32(i)
	}
	sort.Slice(snapshot.QueryInformations, func(i, j int) bool {
		a := snapshot.QueryInformations[i]
		b := snapshot.QueryInformations[j]
		return a.QueryIdx < b.QueryIdx
	})
	sort.Slice(snapshot.QueryStatistics, func(i, j int) bool {
		a := snapshot.QueryStatistics[i]
		b := snapshot.QueryStatistics[j]
		return a.QueryIdx < b.QueryIdx
	})

	// ensure plan references occur in a consistent order
	planRefs := make([]OriginalPlanRef, len(snapshot.QueryPlanReferences))
	for i, planRef := range snapshot.QueryPlanReferences {
		newQueryIdx := slices.IndexFunc(queryRefs, func(item OriginalQueryRef) bool {
			return item.OriginalIdx == planRef.QueryIdx
		})
		planRef.QueryIdx = int32(newQueryIdx)
		planRefs[i] = OriginalPlanRef{
			Original:    planRef,
			OriginalIdx: int32(i),
		}
	}

	sort.Slice(planRefs, func(i, j int) bool {
		return planRefs[i].Original.OriginalPlanId < planRefs[j].Original.OriginalPlanId
	})

	for i, planRef := range planRefs {
		snapshot.QueryPlanReferences[i] = planRef.Original

		pInfoIdx := slices.IndexFunc(snapshot.QueryPlanInformations, func(item *pganalyze_collector.QueryPlanInformation) bool {
			return item.QueryPlanIdx == planRef.OriginalIdx
		})
		pInfo := snapshot.QueryPlanInformations[pInfoIdx]
		pInfo.QueryPlanIdx = int32(i)

		pStatsIdx := slices.IndexFunc(snapshot.QueryPlanStatistics, func(item *pganalyze_collector.QueryPlanStatistic) bool {
			return item.QueryPlanIdx == planRef.OriginalIdx
		})
		pStats := snapshot.QueryPlanStatistics[pStatsIdx]
		pStats.QueryPlanIdx = int32(i)
	}
	sort.Slice(snapshot.QueryPlanInformations, func(i, j int) bool {
		a := snapshot.QueryPlanInformations[i]
		b := snapshot.QueryPlanInformations[j]
		return a.QueryPlanIdx < b.QueryPlanIdx
	})
	sort.Slice(snapshot.QueryPlanStatistics, func(i, j int) bool {
		a := snapshot.QueryPlanStatistics[i]
		b := snapshot.QueryPlanStatistics[j]
		return a.QueryPlanIdx < b.QueryPlanIdx
	})
}
