package selfhosted

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

// GetSystemState - Gets system information about a self-hosted (physical/virtual) system
func GetSystemState(config config.ServerConfig, logger *util.Logger, dataDirectory string) (system state.SystemState) {
	system.Info.Type = state.SelfHostedSystem

	hostInfo, err := host.Info()
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get host information: %s", err)
	} else {
		system.Info.BootTime = time.Unix(int64(hostInfo.BootTime), 0)
		system.Info.SelfHosted = &state.SystemInfoSelfHosted{
			Hostname:        hostInfo.Hostname,
			OperatingSystem: hostInfo.OS,
			Platform:        hostInfo.Platform,
			PlatformFamily:  hostInfo.PlatformFamily,
			PlatformVersion: hostInfo.PlatformVersion,
		}

		if hostInfo.VirtualizationRole == "guest" {
			system.Info.SelfHosted.VirtualizationSystem = hostInfo.VirtualizationSystem
		}

		// TODO:
		//Architecture         string
		//KernelVersion        string
	}

	loadAvg, err := load.Avg()
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get load average: %s", err)
	} else {
		system.Scheduler.Loadavg1min = loadAvg.Load1
		system.Scheduler.Loadavg5min = loadAvg.Load5
		system.Scheduler.Loadavg15min = loadAvg.Load15
	}

	memory, err := mem.VirtualMemory()
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get virtual memory stats: %s", err)
	} else {
		system.Memory.TotalBytes = memory.Total
		system.Memory.CachedBytes = memory.Cached
		system.Memory.BuffersBytes = memory.Buffers
		system.Memory.FreeBytes = memory.Free
		system.Memory.ActiveBytes = memory.Active
		system.Memory.InactiveBytes = memory.Inactive
		system.Memory.AvailableBytes = memory.Available
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get swap stats: %s", err)
	} else {
		system.Memory.SwapUsedBytes = swap.Used
		system.Memory.SwapTotalBytes = swap.Total
	}

	// TODO: Read the stats below from /proc/meminfo (or patch gopsutil to do so)
	system.Memory.WritebackBytes = 0
	system.Memory.DirtyBytes = 0
	system.Memory.SlabBytes = 0
	system.Memory.MappedBytes = 0
	system.Memory.PageTablesBytes = 0
	system.Memory.HugePagesSizeBytes = 0
	system.Memory.HugePagesFree = 0
	system.Memory.HugePagesTotal = 0
	system.Memory.HugePagesReserved = 0
	system.Memory.HugePagesSurplus = 0

	cpuInfos, err := cpu.Info()
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get CPU info: %s", err)
	} else {
		system.CPUInfo.Model = cpuInfos[0].ModelName
		system.CPUInfo.CacheSizeBytes = cpuInfos[0].CacheSize * 1024
		system.CPUInfo.SpeedMhz = cpuInfos[0].Mhz

		physicalIds := make(map[string]bool)
		cores := int32(0)
		for _, cpuInfo := range cpuInfos {
			physicalIds[cpuInfo.PhysicalID] = true
			cores += cpuInfo.Cores
		}
		system.CPUInfo.SocketCount = int32(len(physicalIds))
		system.CPUInfo.PhysicalCoreCount = cores
	}

	cpuStats, err := cpu.Times(true)
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get CPU stats: %s", err)
	} else {
		system.CPUInfo.LogicalCoreCount = int32(len(cpuStats))

		system.CPUStats = make(state.CPUStatisticMap)
		for _, cpuStat := range cpuStats {
			system.CPUStats[cpuStat.CPU] = state.CPUStatistic{
				DiffedOnInput:    false,
				UserSeconds:      cpuStat.User,
				SystemSeconds:    cpuStat.System,
				IdleSeconds:      cpuStat.Idle,
				NiceSeconds:      cpuStat.Nice,
				IowaitSeconds:    cpuStat.Iowait,
				IrqSeconds:       cpuStat.Irq,
				SoftIrqSeconds:   cpuStat.Softirq,
				StealSeconds:     cpuStat.Steal,
				GuestSeconds:     cpuStat.Guest,
				GuestNiceSeconds: cpuStat.GuestNice,
			}
		}
	}

	netStats, err := net.IOCounters(true)
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get network stats: %s", err)
	} else {
		system.NetworkStats = make(state.NetworkStatsMap)
		for _, netStat := range netStats {
			if netStat.BytesRecv == 0 && netStat.BytesSent == 0 {
				continue
			}

			system.NetworkStats[netStat.Name] = state.NetworkStats{
				ReceiveThroughputBytes:  netStat.BytesRecv,
				TransmitThroughputBytes: netStat.BytesSent,
			}
		}
	}

	system.Disks = make(state.DiskMap)
	disks, err := disk.IOCounters()
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get disk I/O stats: %s", err)

		// We need to insert a dummy device, otherwise we can't attach the partitions anywhere
		system.Disks["/"] = state.Disk{}
	} else {
		system.DiskStats = make(state.DiskStatsMap)
		for _, disk := range disks {
			system.Disks[disk.Name] = state.Disk{
			// TODO: DiskType, Scheduler
			}

			system.DiskStats[disk.Name] = state.DiskStats{
				ReadsCompleted:  disk.ReadCount,
				BytesRead:       disk.ReadBytes,
				ReadTimeMs:      disk.ReadTime,
				WritesCompleted: disk.WriteCount,
				BytesWritten:    disk.WriteBytes,
				WriteTimeMs:     disk.WriteTime,
				IoTime:          disk.IoTime,
				// TODO: ReadsMerged, WritesMerged, AvgQueueSize
			}
		}
	}

	xlogDirectory, err := filepath.EvalSymlinks(dataDirectory + "/pg_xlog")
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to resolve xlog path: %s", err)
		xlogDirectory = dataDirectory + "/pg_xlog"
	}

	diskPartitions, err := disk.Partitions(true)
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get disk partitions: %s", err)
	} else {
		system.DiskPartitions = make(state.DiskPartitionMap)
		for _, partition := range diskPartitions {
			// Linux partition types we can ignore
			if partition.Fstype == "devtmpfs" || partition.Fstype == "tmpfs" || partition.Fstype == "devpts" ||
				partition.Fstype == "fusectl" || partition.Fstype == "proc" || partition.Fstype == "cgroup" ||
				partition.Fstype == "securityfs" || partition.Fstype == "debugfs" || partition.Fstype == "sysfs" ||
				partition.Fstype == "pstore" {
				continue
			}

			// OSX partition types we can ignore
			if partition.Fstype == "autofs" || partition.Fstype == "devfs" {
				continue
			}

			diskUsage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				logger.PrintVerbose("Selfhosted/System: Failed to get disk partition usage stats: %s", err)
			} else {
				var diskName string

				for name := range system.Disks {
					if strings.HasPrefix(partition.Device, name) {
						diskName = name
						break
					}
				}

				if strings.HasPrefix(dataDirectory, partition.Mountpoint) && len(system.DataDirectoryPartition) < len(partition.Mountpoint) {
					system.DataDirectoryPartition = partition.Mountpoint
				}
				if strings.HasPrefix(xlogDirectory, partition.Mountpoint) && len(system.XlogPartition) < len(partition.Mountpoint) {
					system.XlogPartition = partition.Mountpoint
				}

				system.DiskPartitions[partition.Mountpoint] = state.DiskPartition{
					DiskName:       diskName,
					PartitionName:  partition.Device,
					FilesystemType: partition.Fstype,
					FilesystemOpts: partition.Opts,
					UsedBytes:      diskUsage.Total - diskUsage.Free,
					TotalBytes:     diskUsage.Total,
				}
			}
		}
	}

	// TODO: Locate the filesystem that the data directory lives on

	return
}
