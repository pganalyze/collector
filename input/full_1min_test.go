package input

import (
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/pganalyze/collector/state"
)

func TestDiffStatements(t *testing.T) {
	created := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	recreated := created.Add(30 * time.Second)
	key := state.PostgresStatementKey{DatabaseOid: 1, UserOid: 10, QueryID: 123, Toplevel: true}
	otherKey := state.PostgresStatementKey{DatabaseOid: 1, UserOid: 10, QueryID: 456, Toplevel: true}

	tests := []struct {
		name      string
		prev      state.PostgresStatementStats
		curr      state.PostgresStatementStats
		wantCalls int64 // 0 means the statement is not part of the diff
		wantTime  float64
	}{
		{
			name:      "same entry",
			prev:      state.PostgresStatementStats{Calls: 1000, TotalTime: 500, StatsSince: null.TimeFrom(created)},
			curr:      state.PostgresStatementStats{Calls: 1050, TotalTime: 520, StatsSince: null.TimeFrom(created)},
			wantCalls: 50,
			wantTime:  20,
		},
		{
			name: "same entry without activity",
			prev: state.PostgresStatementStats{Calls: 1000, TotalTime: 500, StatsSince: null.TimeFrom(created)},
			curr: state.PostgresStatementStats{Calls: 1000, TotalTime: 500, StatsSince: null.TimeFrom(created)},
		},
		{
			name:      "recreated entry with fewer calls",
			prev:      state.PostgresStatementStats{Calls: 1000, TotalTime: 500, StatsSince: null.TimeFrom(created)},
			curr:      state.PostgresStatementStats{Calls: 50, TotalTime: 20, StatsSince: null.TimeFrom(recreated)},
			wantCalls: 50,
			wantTime:  20,
		},
		{
			name:      "recreated entry with more calls",
			prev:      state.PostgresStatementStats{Calls: 1000, TotalTime: 500, StatsSince: null.TimeFrom(created)},
			curr:      state.PostgresStatementStats{Calls: 1500, TotalTime: 300, StatsSince: null.TimeFrom(recreated)},
			wantCalls: 1500,
			wantTime:  300,
		},
		{
			name:      "recreated entry with the same calls",
			prev:      state.PostgresStatementStats{Calls: 1, TotalTime: 5000, StatsSince: null.TimeFrom(created)},
			curr:      state.PostgresStatementStats{Calls: 1, TotalTime: 4000, StatsSince: null.TimeFrom(recreated)},
			wantCalls: 1,
			wantTime:  4000,
		},
		{
			name: "older pg_stat_statements: fewer calls are dropped",
			prev: state.PostgresStatementStats{Calls: 1000, TotalTime: 500},
			curr: state.PostgresStatementStats{Calls: 50, TotalTime: 20},
		},
		{
			name:      "older pg_stat_statements: diffed by key alone",
			prev:      state.PostgresStatementStats{Calls: 1000, TotalTime: 500},
			curr:      state.PostgresStatementStats{Calls: 1500, TotalTime: 700},
			wantCalls: 500,
			wantTime:  200,
		},
		{
			name:      "previous statistics saved without stats_since",
			prev:      state.PostgresStatementStats{Calls: 1000, TotalTime: 500},
			curr:      state.PostgresStatementStats{Calls: 1050, TotalTime: 520, StatsSince: null.TimeFrom(created)},
			wantCalls: 50,
			wantTime:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := state.PostgresStatementStatsMap{key: tt.prev}
			curr := state.PostgresStatementStatsMap{key: tt.curr}
			diff := diffStatements(curr, prev)

			got, exists := diff[key]
			if tt.wantCalls == 0 {
				if exists {
					t.Fatalf("expected statement to be left out, got %+v", got)
				}
				return
			}
			if !exists {
				t.Fatalf("expected statement in diff, but it was left out")
			}
			if got.Calls != tt.wantCalls || got.TotalTime != tt.wantTime {
				t.Errorf("got calls=%d total_time=%v, want calls=%d total_time=%v", got.Calls, got.TotalTime, tt.wantCalls, tt.wantTime)
			}
		})
	}

	t.Run("new statement", func(t *testing.T) {
		prev := state.PostgresStatementStatsMap{otherKey: {Calls: 1}}
		curr := state.PostgresStatementStatsMap{otherKey: {Calls: 1}, key: {Calls: 7, StatsSince: null.TimeFrom(created)}}
		diff := diffStatements(curr, prev)
		if diff[key].Calls != 7 {
			t.Errorf("got calls=%d, want 7", diff[key].Calls)
		}
		if _, exists := diff[otherKey]; exists {
			t.Errorf("expected statement without activity to be left out")
		}
	})
}

func TestDiffPlanStats(t *testing.T) {
	statementKey := state.PostgresStatementKey{DatabaseOid: 1, UserOid: 10, QueryID: 123, Toplevel: true}
	key := state.PostgresPlanKey{PostgresStatementKey: statementKey, PlanID: 1}
	otherKey := state.PostgresPlanKey{PostgresStatementKey: statementKey, PlanID: 2}

	t.Run("existing plan", func(t *testing.T) {
		prev := state.PostgresPlanStatsMap{key: {Calls: 1000, TotalTime: 500}}
		curr := state.PostgresPlanStatsMap{key: {Calls: 1050, TotalTime: 520}}
		diff := diffPlanStats(curr, prev)
		if got := diff[key]; got.Calls != 50 || got.TotalTime != 20 {
			t.Errorf("got calls=%d total_time=%v, want calls=50 total_time=20", got.Calls, got.TotalTime)
		}
	})

	t.Run("existing plan without activity", func(t *testing.T) {
		prev := state.PostgresPlanStatsMap{key: {Calls: 1000, TotalTime: 500}}
		curr := state.PostgresPlanStatsMap{key: {Calls: 1000, TotalTime: 500}}
		diff := diffPlanStats(curr, prev)
		if got, exists := diff[key]; exists {
			t.Errorf("expected plan to be left out, got %+v", got)
		}
	})

	t.Run("new plan", func(t *testing.T) {
		prev := state.PostgresPlanStatsMap{otherKey: {Calls: 1}}
		curr := state.PostgresPlanStatsMap{otherKey: {Calls: 1}, key: {Calls: 7, TotalTime: 3}}
		diff := diffPlanStats(curr, prev)
		if got := diff[key]; got.Calls != 7 || got.TotalTime != 3 {
			t.Errorf("got calls=%d total_time=%v, want calls=7 total_time=3", got.Calls, got.TotalTime)
		}
		if _, exists := diff[otherKey]; exists {
			t.Errorf("expected plan without activity to be left out")
		}
	})

	// Plans are diffed by key alone, since plan statistics have no creation time
	t.Run("fewer calls than before", func(t *testing.T) {
		prev := state.PostgresPlanStatsMap{key: {Calls: 1000, TotalTime: 500}}
		curr := state.PostgresPlanStatsMap{key: {Calls: 50, TotalTime: 20}}
		diff := diffPlanStats(curr, prev)
		if got, exists := diff[key]; exists {
			t.Errorf("expected plan to be left out, got %+v", got)
		}
	})

	t.Run("no reference point", func(t *testing.T) {
		curr := state.PostgresPlanStatsMap{key: {Calls: 7, TotalTime: 3}}
		for _, prev := range []state.PostgresPlanStatsMap{nil, {}} {
			diff := diffPlanStats(curr, prev)
			if diff == nil {
				t.Fatalf("expected an empty map, got nil")
			}
			if len(diff) != 0 {
				t.Errorf("expected no plans in diff, got %+v", diff)
			}
		}
	})
}

func TestDiffServerIoStats(t *testing.T) {
	key := state.PostgresServerIoStatsKey{BackendType: "client backend", IoObject: "relation", IoContext: "normal"}
	otherKey := state.PostgresServerIoStatsKey{BackendType: "checkpointer", IoObject: "relation", IoContext: "normal"}

	t.Run("existing entry", func(t *testing.T) {
		prev := state.PostgresServerIoStatsMap{key: {Reads: 1000, ReadTime: 500, Hits: 20000}}
		curr := state.PostgresServerIoStatsMap{key: {Reads: 1050, ReadTime: 520, Hits: 20300}}
		diff := diffServerIoStats(curr, prev)
		if got := diff[key]; got.Reads != 50 || got.ReadTime != 20 || got.Hits != 300 {
			t.Errorf("got reads=%d read_time=%v hits=%d, want reads=50 read_time=20 hits=300", got.Reads, got.ReadTime, got.Hits)
		}
	})

	t.Run("existing entry without activity", func(t *testing.T) {
		prev := state.PostgresServerIoStatsMap{key: {Reads: 1000, ReadTime: 500, Hits: 20000}}
		curr := state.PostgresServerIoStatsMap{key: {Reads: 1000, ReadTime: 500, Hits: 20000}}
		diff := diffServerIoStats(curr, prev)
		if got, exists := diff[key]; exists {
			t.Errorf("expected entry to be left out, got %+v", got)
		}
	})

	t.Run("activity in a single counter", func(t *testing.T) {
		prev := state.PostgresServerIoStatsMap{key: {Reads: 1000, Fsyncs: 3}}
		curr := state.PostgresServerIoStatsMap{key: {Reads: 1000, Fsyncs: 4}}
		diff := diffServerIoStats(curr, prev)
		if got := diff[key]; got.Fsyncs != 1 || got.Reads != 0 {
			t.Errorf("got fsyncs=%d reads=%d, want fsyncs=1 reads=0", got.Fsyncs, got.Reads)
		}
	})

	t.Run("new entry", func(t *testing.T) {
		prev := state.PostgresServerIoStatsMap{otherKey: {Writes: 1}}
		curr := state.PostgresServerIoStatsMap{otherKey: {Writes: 1}, key: {Reads: 7, ReadTime: 3}}
		diff := diffServerIoStats(curr, prev)
		if got := diff[key]; got.Reads != 7 || got.ReadTime != 3 {
			t.Errorf("got reads=%d read_time=%v, want reads=7 read_time=3", got.Reads, got.ReadTime)
		}
		if _, exists := diff[otherKey]; exists {
			t.Errorf("expected entry without activity to be left out")
		}
	})

	t.Run("new entry that never saw activity", func(t *testing.T) {
		prev := state.PostgresServerIoStatsMap{otherKey: {Writes: 1}}
		curr := state.PostgresServerIoStatsMap{otherKey: {Writes: 1}, key: {}}
		diff := diffServerIoStats(curr, prev)
		if got, exists := diff[key]; exists {
			t.Errorf("expected entry to be left out, got %+v", got)
		}
	})

	t.Run("no reference point", func(t *testing.T) {
		curr := state.PostgresServerIoStatsMap{key: {Reads: 7, ReadTime: 3}}
		for _, prev := range []state.PostgresServerIoStatsMap{nil, {}} {
			diff := diffServerIoStats(curr, prev)
			if diff == nil {
				t.Fatalf("expected an empty map, got nil")
			}
			if len(diff) != 0 {
				t.Errorf("expected no entries in diff, got %+v", diff)
			}
		}
	})
}
