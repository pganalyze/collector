package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformCollectorStats(s snapshot.FullSnapshot, newState state.State, diffState state.DiffState) snapshot.FullSnapshot {
	s.CollectorStatistic = &snapshot.CollectorStatistic{
		GoVersion:                diffState.CollectorStats.GoVersion,
		MemoryHeapAllocatedBytes: diffState.CollectorStats.MemoryHeapAllocatedBytes,
		MemoryHeapObjects:        diffState.CollectorStats.MemoryHeapObjects,
		MemorySystemBytes:        diffState.CollectorStats.MemorySystemBytes,
		MemoryRssBytes:           diffState.CollectorStats.MemoryRssBytes,
		ActiveGoroutines:         diffState.CollectorStats.ActiveGoroutines,
		CgoCalls:                 diffState.CollectorStats.CgoCalls,
	}
	return s
}
