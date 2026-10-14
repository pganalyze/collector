package runner

import (
	"slices"
	"testing"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
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
