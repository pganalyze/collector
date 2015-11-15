package systemstats

// Memory - Metrics related to system memory
type Memory struct {
	ApplicationsBytes *int64 `json:"applications_bytes,omitempty"`
	BuffersBytes      *int64 `json:"buffers_bytes,omitempty"`
	DirtyBytes        *int64 `json:"dirty_bytes,omitempty"`
	FreeBytes         *int64 `json:"free_bytes,omitempty"`
	PagecacheBytes    *int64 `json:"pagecache_bytes,omitempty"`
	SwapFreeBytes     *int64 `json:"swap_free_bytes,omitempty"`
	SwapTotalBytes    *int64 `json:"swap_total_bytes,omitempty"`
	TotalBytes        *int64 `json:"total_bytes,omitempty"`
	WritebackBytes    *int64 `json:"writeback_bytes,omitempty"`
}
