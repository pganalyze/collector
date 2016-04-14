package systemstats

import "gopkg.in/guregu/null.v2"

// Storage - Information about the storage used by the database
type Storage struct {
	BytesAvailable *int64  `json:"bytes_available"`
	BytesTotal     *int64  `json:"bytes_total"`
	Mountpoint     *string `json:"mountpoint,omitempty"`
	Name           *string `json:"name,omitempty"`
	Path           *string `json:"path,omitempty"`

	Perfdata StoragePerfdata `json:"perfdata"`
}

// StoragePerfdata - Metrics gathered about the underlying storage
type StoragePerfdata struct {
	// 0 = counters, raw data
	// 1 = diff, (only) useful data
	Version int `json:"version"`

	// Version 0/1
	ReadIops       *int64   `json:"rd_ios"`       // (count/sec)
	WriteIops      *int64   `json:"wr_ios"`       // (count/sec)
	IopsInProgress *int64   `json:"ios_in_prog"`  // (count)
	AvgReqSize     null.Int `json:"avg_req_size"` // (avg)

	// Version 1 only
	ReadLatency     *float64 `json:"rd_latency,omitempty"`    // (avg seconds)
	ReadThroughput  *int64   `json:"rd_throughput,omitempty"` // (bytes/sec)
	WriteLatency    *float64 `json:"wr_latency,omitempty"`    // (avg seconds)
	WriteThroughput *int64   `json:"wr_throughput,omitempty"` // (bytes/sec)

	// Version 0 only
	ReadMerges   *int64 `json:"rd_merges,omitempty"`
	ReadSectors  *int64 `json:"rd_sectors,omitempty,omitempty"`
	ReadTicks    *int64 `json:"rd_ticks,omitempty"`
	WriteMerges  *int64 `json:"wr_merges,omitempty"`
	WriteSectors *int64 `json:"wr_sectors,omitempty"`
	WriteTicks   *int64 `json:"wr_ticks,omitempty"`
	TotalTicks   *int64 `json:"tot_ticks,omitempty"`
	RequestTicks *int64 `json:"rq_ticks,omitempty"`
}
