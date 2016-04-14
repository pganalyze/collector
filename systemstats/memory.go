package systemstats

import "gopkg.in/guregu/null.v2"

// Memory - Metrics related to system memory
type Memory struct {
	ApplicationsBytes null.Int `json:"applications_bytes,omitempty"`
	BuffersBytes      null.Int `json:"buffers_bytes,omitempty"`
	DirtyBytes        null.Int `json:"dirty_bytes,omitempty"`
	FreeBytes         null.Int `json:"free_bytes,omitempty"`
	PagecacheBytes    null.Int `json:"pagecache_bytes,omitempty"`
	SwapFreeBytes     null.Int `json:"swap_free_bytes,omitempty"`
	SwapTotalBytes    null.Int `json:"swap_total_bytes,omitempty"`
	TotalBytes        null.Int `json:"total_bytes,omitempty"`
	WritebackBytes    null.Int `json:"writeback_bytes,omitempty"`
	ActiveBytes       null.Int `json:"active_bytes,omitempty"`
}
