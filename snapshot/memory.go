//go:generate msgp

package snapshot

// Memory - Metrics related to system memory
type Memory struct {
	ApplicationsBytes NullableInt `msg:"applications_bytes,omitempty"`
	BuffersBytes      NullableInt `msg:"buffers_bytes,omitempty"`
	DirtyBytes        NullableInt `msg:"dirty_bytes,omitempty"`
	FreeBytes         NullableInt `msg:"free_bytes,omitempty"`
	PagecacheBytes    NullableInt `msg:"pagecache_bytes,omitempty"`
	SwapFreeBytes     NullableInt `msg:"swap_free_bytes,omitempty"`
	SwapTotalBytes    NullableInt `msg:"swap_total_bytes,omitempty"`
	TotalBytes        NullableInt `msg:"total_bytes,omitempty"`
	WritebackBytes    NullableInt `msg:"writeback_bytes,omitempty"`
	ActiveBytes       NullableInt `msg:"active_bytes,omitempty"`
}
