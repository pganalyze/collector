package state

import (
	"sync/atomic"
)

// MemoryLimit tracks a shared byte counter against a configurable cap.
// Callers use Add/Remove to adjust the counter and Size to inspect it.
// OverLimit reports whether the current usage has exceeded the cap.
type MemoryLimit struct {
	bytes atomic.Int64
	limit int64
}

// Global memory limit for all snapshot queues
var QueueMemory = NewMemoryLimit(200 * 1024 * 1024)

func NewMemoryLimit(cap int64) *MemoryLimit {
	return &MemoryLimit{limit: cap}
}

func (m *MemoryLimit) Add(n int64) int64 {
	return m.bytes.Add(n)
}

func (m *MemoryLimit) Remove(n int64) int64 {
	return m.bytes.Add(-n)
}

func (m *MemoryLimit) Size() int64 {
	return m.bytes.Load()
}

func (m *MemoryLimit) OverLimit() bool {
	return m.bytes.Load() > m.limit
}
