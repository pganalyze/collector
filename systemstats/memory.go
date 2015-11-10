package systemstats

type Memory struct {
	ApplicationsBytes *int64 `json:"applications_bytes"`
	BuffersBytes      *int64 `json:"buffers_bytes"`
	DirtyBytes        *int64 `json:"dirty_bytes"`
	FreeBytes         *int64 `json:"free_bytes"`
	PagecacheBytes    *int64 `json:"pagecache_bytes"`
	SwapFreeBytes     *int64 `json:"swap_free_bytes"`
	SwapTotalBytes    *int64 `json:"swap_total_bytes"`
	TotalBytes        *int64 `json:"total_bytes"`
	WritebackBytes    *int64 `json:"writeback_bytes"`
}
