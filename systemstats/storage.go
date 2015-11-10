package systemstats

import (
	null "gopkg.in/guregu/null.v2"
)

type Storage struct {
	BytesAvailable *int64      `json:"bytes_available"`
	BytesTotal     *int64      `json:"bytes_total"`
	Mountpoint     null.String `json:"mountpoint"`
	Name           null.String `json:"name"`
	Path           null.String `json:"path"`
	Encrypted      *bool       `json:"encrypted"`

	Perfdata StoragePerfdata `json:"perfdata"`
}

type StoragePerfdata struct {
	// 0 = counters, raw data
	// 1 = diff, (only) useful data
	Version int `json:"version"`

	// Version 0/1
	ReadIops       *int64 `json:"rd_ios"`      // (count/sec)
	WriteIops      *int64 `json:"wr_ios"`      // (count/sec)
	IopsInProgress *int64 `json:"ios_in_prog"` // (count/sec)

	// Version 1 only
	ReadLatency     *float64 `json:"rd_latency"`    // (avg seconds)
	ReadThroughput  *int64   `json:"rd_throughput"` // (bytes/sec)
	WriteLatency    *float64 `json:"wr_latency"`    // (avg seconds)
	WriteThroughput *int64   `json:"wr_throughput"` // (bytes/sec)

	// Version 0 only
	ReadMerges   *int64 `json:"rd_merges"`
	ReadSectors  *int64 `json:"rd_sectors"`
	ReadTicks    *int64 `json:"rd_ticks"`
	WriteMerges  *int64 `json:"wr_merges"`
	WriteSectors *int64 `json:"wr_sectors"`
	WriteTicks   *int64 `json:"wr_ticks"`
	TotalTicks   *int64 `json:"tot_ticks"`
	RequestTicks *int64 `json:"rq_ticks"`
}
