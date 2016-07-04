package state

import (
	"time"

	"gopkg.in/guregu/null.v3"
)

// SystemState - All kinds of system-related information and metrics
type SystemState struct {
	Info         SystemInfo
	Scheduler    Scheduler
	Memory       Memory
	CPUInfo      CPUInformation
	CPUStats     CPUStatisticMap
	NetworkStats NetworkStatsMap
}

// SystemType - Enum that describes which kind of system we're monitoring
type SystemType int

// Treat this list as append-only and never change the order
const (
	SelfHostedSystem SystemType = iota
	AmazonRdsSystem
	HerokuSystem
)

type SystemInfo struct {
	Type SystemType

	SelfHosted *SystemInfoSelfHosted
	AmazonRds  *SystemInfoAmazonRds

	BootTime time.Time
}

// SystemInfoSelfHosted - System information for self-hosted systems (both physical and virtual)
type SystemInfoSelfHosted struct {
	Hostname             string
	Architecture         string
	OperatingSystem      string
	Platform             string
	PlatformFamily       string
	PlatformVersion      string
	VirtualizationSystem string // Name of the virtualization system (only if we're a guest)
	KernelVersion        string
}

// SystemInfoAmazonRds - System information for Amazon RDS systems
type SystemInfoAmazonRds struct {
	Region                     string
	InstanceClass              string
	InstanceID                 string
	Status                     string
	AvailabilityZone           string
	PubliclyAccessible         bool
	MultiAz                    bool
	SecondaryAvailabilityZone  string
	CaCertificate              string
	AutoMinorVersionUpgrade    bool
	AutoMajorVersionUpgrade    bool
	PreferredMaintenanceWindow string
	PreferredBackupWindow      string
	LatestRestorableTime       time.Time
	BackupRetentionPeriodDays  int32
	MasterUsername             string
	InitialDbName              string
	CreatedAt                  time.Time
	StorageProvisionedIOPS     int32
	StorageAllocatedGigabytes  int32
	StorageEncrypted           bool
	StorageType                string
	EnhancedMonitoring         bool
	ParameterApplyStatus       string
	ParameterPgssEnabled       bool
}

// Scheduler - Information about the OS scheduler
type Scheduler struct {
	Loadavg1min  float64
	Loadavg5min  float64
	Loadavg15min float64
}

// Memory - Metrics related to system memory
type Memory struct {
	TotalBytes      uint64
	CachedBytes     uint64
	BuffersBytes    uint64
	FreeBytes       uint64
	WritebackBytes  uint64
	DirtyBytes      uint64
	SlabBytes       uint64
	MappedBytes     uint64
	PageTablesBytes uint64
	ActiveBytes     uint64
	InactiveBytes   uint64
	AvailableBytes  uint64
	SwapUsedBytes   uint64
	SwapTotalBytes  uint64

	HugePagesSizeBytes uint64
	HugePagesFree      uint64
	HugePagesTotal     uint64
	HugePagesReserved  uint64
	HugePagesSurplus   uint64
}

type CPUInformation struct {
	Model             string
	CacheSizeBytes    int32
	SpeedMhz          float64
	SocketCount       int32
	PhysicalCoreCount int32
	LogicalCoreCount  int32
}

// CPUStatisticMap - Map of all CPU statistics (Key = CPU ID)
type CPUStatisticMap map[string]CPUStatistic

// CPUStatistic - Statistics for a single CPU core
type CPUStatistic struct {
	MeasuredAsSeconds bool // True if this uses the Seconds counters, false if it uses percentages

	// Seconds (counter values that need to be diff-ed between runs)
	UserSeconds      float64
	SystemSeconds    float64
	IdleSeconds      float64
	NiceSeconds      float64
	IowaitSeconds    float64
	IrqSeconds       float64
	SoftIrqSeconds   float64
	StealSeconds     float64
	GuestSeconds     float64
	GuestNiceSeconds float64

	// Percentages (don't need to be diff-ed)
	Percentages *DiffedSystemCPUStats
}

// DiffedSystemCPUStatsMap - Map of all CPU statistics (Key = CPU ID)
type DiffedSystemCPUStatsMap map[string]DiffedSystemCPUStats

// DiffedSystemCPUStats - CPU statistics as percentages
type DiffedSystemCPUStats struct {
	UserPercent      float64
	SystemPercent    float64
	IdlePercent      float64
	NicePercent      float64
	IowaitPercent    float64
	IrqPercent       float64
	SoftIrqPercent   float64
	StealPercent     float64
	GuestPercent     float64
	GuestNicePercent float64
}

func (curr CPUStatistic) DiffSince(prev CPUStatistic) DiffedSystemCPUStats {
	userSecs := curr.UserSeconds - prev.UserSeconds
	systemSecs := curr.SystemSeconds - prev.SystemSeconds
	idleSecs := curr.IdleSeconds - prev.IdleSeconds
	niceSecs := curr.NiceSeconds - prev.NiceSeconds
	iowaitSecs := curr.IowaitSeconds - prev.IowaitSeconds
	irqSecs := curr.IrqSeconds - prev.IrqSeconds
	softIrqSecs := curr.SoftIrqSeconds - prev.SoftIrqSeconds
	stealSecs := curr.StealSeconds - prev.StealSeconds
	guestSecs := curr.GuestSeconds - prev.GuestSeconds
	guestNiceSecs := curr.GuestNiceSeconds - prev.GuestNiceSeconds
	totalSecs := userSecs + systemSecs + idleSecs + niceSecs + iowaitSecs + irqSecs + softIrqSecs + stealSecs + guestSecs + guestNiceSecs

	return DiffedSystemCPUStats{
		UserPercent:      userSecs / totalSecs * 100,
		SystemPercent:    systemSecs / totalSecs * 100,
		IdlePercent:      idleSecs / totalSecs * 100,
		NicePercent:      niceSecs / totalSecs * 100,
		IowaitPercent:    iowaitSecs / totalSecs * 100,
		IrqPercent:       irqSecs / totalSecs * 100,
		SoftIrqPercent:   softIrqSecs / totalSecs * 100,
		StealPercent:     stealSecs / totalSecs * 100,
		GuestPercent:     guestSecs / totalSecs * 100,
		GuestNicePercent: guestNiceSecs / totalSecs * 100,
	}
}

// NetworkStatsMap - Map of all network statistics (Key = Interface Name)
type NetworkStatsMap map[string]NetworkStats

// NetworkStats - Information about the network activity on a single interface
type NetworkStats struct {
	ReceiveThroughputBytes  uint64
	TransmitThroughputBytes uint64
}

// DiffedNetworkStats - Network statistics for a single interface as a diff
type DiffedNetworkStats NetworkStats

// DiffedNetworkStatsMap - Map of network statistics as a diff (Key = Interface Name)
type DiffedNetworkStatsMap map[string]DiffedNetworkStats

// DiffSince - Calculate the diff between two network stats runs
func (curr NetworkStats) DiffSince(prev NetworkStats) DiffedNetworkStats {
	return DiffedNetworkStats{
		ReceiveThroughputBytes:  curr.ReceiveThroughputBytes - prev.ReceiveThroughputBytes,
		TransmitThroughputBytes: curr.TransmitThroughputBytes - prev.TransmitThroughputBytes,
	}
}

// ---

// Storage - Information about the storage used by the database
type Storage struct {
	BytesAvailable *int64
	BytesTotal     *int64
	Mountpoint     *string
	Name           *string
	Path           *string

	Perfdata StoragePerfdata
}

// StoragePerfdata - Metrics gathered about the underlying storage
type StoragePerfdata struct {
	// 0 = counters, raw data
	// 1 = diff, (only) useful data
	Version int

	// Version 0/1
	ReadIops       *int64   // (count/sec)
	WriteIops      *int64   // (count/sec)
	IopsInProgress *int64   // (count)
	AvgReqSize     null.Int // (avg)

	// Version 1 only
	ReadLatency     *float64 // (avg seconds)
	ReadThroughput  *int64   // (bytes/sec)
	WriteLatency    *float64 // (avg seconds)
	WriteThroughput *int64   // (bytes/sec)

	// Version 0 only
	ReadMerges   *int64
	ReadSectors  *int64
	ReadTicks    *int64
	WriteMerges  *int64
	WriteSectors *int64
	WriteTicks   *int64
	TotalTicks   *int64
	RequestTicks *int64
}
