package state

import "gopkg.in/guregu/null.v3"

// SystemSnapshot - All kinds of system-related information and metrics
type SystemState struct {
	SystemType SystemType  `json:"system_type"`
	SystemInfo interface{} `json:"system_info,omitempty"`
	Storage    []Storage   `json:"storage"`
	CPU        CPU         `json:"cpu"`
	Memory     Memory      `json:"memory"`
	Network    *Network    `json:"network,omitempty"`
	Scheduler  Scheduler   `json:"scheduler,omitempty"`
}

// SystemType - Enum that describes which kind of system we're monitoring
type SystemType int

// Treat this list as append-only and never change the order
const (
	PhysicalSystem SystemType = iota
	VirtualSystem
	AmazonRdsSystem
	HerokuSystem
)

// Network - Information about the network activity going in and out of the database
type Network struct {
	ReceiveThroughput  *int64 `json:"receive_throughput"`
	TransmitThroughput *int64 `json:"transmit_throughput"`
}

// CPU - Information about CPU activity
type CPU struct {
	Utilization *float64 `json:"utilization"`

	BusyTimesGuestMsec     null.Int `json:"busy_times_guest_msec,omitempty"`
	BusyTimesGuestNiceMsec null.Int `json:"busy_times_guest_nice_msec,omitempty"`
	BusyTimesIdleMsec      null.Int `json:"busy_times_idle_msec,omitempty"`
	BusyTimesIowaitMsec    null.Int `json:"busy_times_iowait_msec,omitempty"`
	BusyTimesIrqMsec       null.Int `json:"busy_times_irq_msec,omitempty"`
	BusyTimesNiceMsec      null.Int `json:"busy_times_nice_msec,omitempty"`
	BusyTimesSoftirqMsec   null.Int `json:"busy_times_softirq_msec,omitempty"`
	BusyTimesStealMsec     null.Int `json:"busy_times_steal_msec,omitempty"`
	BusyTimesSystemMsec    null.Int `json:"busy_times_system_msec,omitempty"`
	BusyTimesUserMsec      null.Int `json:"busy_times_user_msec,omitempty"`

	HardwareCacheSize      *string    `json:"hardware_cache_size,omitempty"`
	HardwareModel          *string    `json:"hardware_model,omitempty"`
	HardwareSockets        null.Int   `json:"hardware_sockets,omitempty"`
	HardwareCoresPerSocket null.Int   `json:"hardware_cores_per_socket,omitempty"`
	HardwareSpeedMhz       null.Float `json:"hardware_speed_mhz,omitempty"`
}

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

// Scheduler - Information about the OS scheduler
type Scheduler struct {
	ContextSwitches null.Int `json:"context_switches,omitempty"`
	Interrupts      null.Int `json:"interrupts,omitempty"`

	Loadavg1min  null.Float `json:"loadavg_1min,omitempty"`
	Loadavg5min  null.Float `json:"loadavg_5min,omitempty"`
	Loadavg15min null.Float `json:"loadavg_15min,omitempty"`

	ProcsBlocked null.Int `json:"procs_blocked,omitempty"`
	ProcsCreated null.Int `json:"procs_created,omitempty"`
	ProcsRunning null.Int `json:"procs_running,omitempty"`
}

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
