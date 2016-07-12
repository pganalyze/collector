package input

import (
	"os"
	"runtime"

	"github.com/pganalyze/collector/state"
	"github.com/shirou/gopsutil/process"
)

func getMemoryRssBytes() uint64 {
	pid := os.Getpid()

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return 0
	}

	mem, err := p.MemoryInfo()
	if err != nil {
		return 0
	}

	return mem.RSS
}

func getCollectorStats() state.CollectorStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return state.CollectorStats{
		ActiveGoroutines:         int32(runtime.NumGoroutine()),
		CgoCalls:                 runtime.NumCgoCall(),
		MemoryHeapAllocatedBytes: memStats.HeapAlloc,
		MemoryHeapObjects:        memStats.HeapObjects,
		MemorySystemBytes:        memStats.Sys,
		MemoryRssBytes:           getMemoryRssBytes(),
	}
}
