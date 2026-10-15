package runner

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func TestPendingQueryRunsOrder(t *testing.T) {
	server := state.MakeServer(config.ServerConfig{}, false)
	var want []int64
	// Enough runs that random map iteration order is near-certain to be unsorted
	for id := int64(1); id <= 50; id++ {
		run := &state.QueryRun{Id: id}
		if id%5 == 0 {
			run.FinishedAt = time.Now()
		} else {
			want = append(want, id)
		}
		server.QueryRuns[id] = run
	}

	var got []int64
	for _, run := range pendingQueryRuns(server) {
		got = append(got, run.Id)
	}

	if !slices.Equal(got, want) {
		t.Errorf("pendingQueryRuns() returned IDs %v, want %v", got, want)
	}
}

func TestCancelQueryRuns(t *testing.T) {
	server := state.MakeServer(config.ServerConfig{}, false)
	queued := &state.QueryRun{Id: 1}
	running := &state.QueryRun{Id: 2}
	finishedAt := time.Now().Add(-time.Minute)
	finished := &state.QueryRun{Id: 3, FinishedAt: finishedAt, Result: "plan"}
	for _, run := range []*state.QueryRun{queued, running, finished} {
		server.QueryRuns[run.Id] = run
	}
	runCtx, started := startQueryRun(context.Background(), server, running)
	if !started {
		t.Fatalf("startQueryRun() did not start run 2")
	}

	// 4 is unknown, and 1 is canceled twice
	cancelQueryRuns(server, []int64{1, 2, 3, 4, 1}, &util.Logger{})

	for _, run := range []*state.QueryRun{queued, running} {
		if run.FinishedAt.IsZero() || run.Error != "Query run was canceled" {
			t.Errorf("run %d: got FinishedAt %v, Error %q, want it finished as canceled", run.Id, run.FinishedAt, run.Error)
		}
	}
	if runCtx.Err() == nil {
		t.Errorf("context of running run was not canceled")
	}
	if finished.FinishedAt != finishedAt || finished.Error != "" || finished.Result != "plan" {
		t.Errorf("finished run was modified: %+v", finished)
	}
	if len(server.QueryRuns) != 3 {
		t.Errorf("got %d query runs after canceling an unknown ID, want 3", len(server.QueryRuns))
	}
	if pending := pendingQueryRuns(server); len(pending) != 0 {
		t.Errorf("got %d pending query runs after canceling, want 0", len(pending))
	}
}

func TestStartQueryRunCanceledAfterSnapshot(t *testing.T) {
	server := state.MakeServer(config.ServerConfig{}, false)
	server.QueryRuns[1] = &state.QueryRun{Id: 1}
	pending := pendingQueryRuns(server)

	cancelQueryRuns(server, []int64{1}, &util.Logger{})

	if _, started := startQueryRun(context.Background(), server, pending[0]); started {
		t.Errorf("startQueryRun() started a run canceled after the pending snapshot")
	}
	if !pending[0].StartedAt.IsZero() {
		t.Errorf("canceled run has StartedAt set")
	}
}
