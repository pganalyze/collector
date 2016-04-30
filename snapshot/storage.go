//go:generate msgp

package snapshot

// Storage - Information about the storage used by the database
type Storage struct {
	BytesAvailable NullableInt    `msg:"bytes_available"`
	BytesTotal     NullableInt    `msg:"bytes_total"`
	Mountpoint     NullableString `msg:"mountpoint,omitempty"`
	Name           NullableString `msg:"name,omitempty"`
	Path           NullableString `msg:"path,omitempty"`

	Perfdata StoragePerfdata `msg:"perfdata"`
}

// StoragePerfdata - Metrics gathered about the underlying storage
type StoragePerfdata struct {
	// 0 = counters, raw data
	// 1 = diff, (only) useful data
	Version int `msg:"version"`

	// Version 0/1
	ReadIops       NullableInt `msg:"rd_ios"`       // (count/sec)
	WriteIops      NullableInt `msg:"wr_ios"`       // (count/sec)
	IopsInProgress NullableInt `msg:"ios_in_prog"`  // (count)
	AvgReqSize     NullableInt `msg:"avg_req_size"` // (avg)

	// Version 1 only
	ReadLatency     NullableFloat `msg:"rd_latency,omitempty"`    // (avg seconds)
	ReadThroughput  NullableInt   `msg:"rd_throughput,omitempty"` // (bytes/sec)
	WriteLatency    NullableFloat `msg:"wr_latency,omitempty"`    // (avg seconds)
	WriteThroughput NullableInt   `msg:"wr_throughput,omitempty"` // (bytes/sec)

	// Version 0 only
	ReadMerges   NullableInt `msg:"rd_merges,omitempty"`
	ReadSectors  NullableInt `msg:"rd_sectors,omitempty,omitempty"`
	ReadTicks    NullableInt `msg:"rd_ticks,omitempty"`
	WriteMerges  NullableInt `msg:"wr_merges,omitempty"`
	WriteSectors NullableInt `msg:"wr_sectors,omitempty"`
	WriteTicks   NullableInt `msg:"wr_ticks,omitempty"`
	TotalTicks   NullableInt `msg:"tot_ticks,omitempty"`
	RequestTicks NullableInt `msg:"rq_ticks,omitempty"`
}
