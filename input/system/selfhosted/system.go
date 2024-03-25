package selfhosted

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/prometheus/procfs"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

type helperStatus struct {
	PostmasterPid    int
	DataDirectory    string
	XlogDirectory    string
	XlogUsedBytes    uint64
	SystemIdentifier string
}

// GetSystemState - Gets system information about a self-hosted (physical/virtual) system
func GetSystemState(server *state.Server, logger *util.Logger) (system state.SystemState) {
	config := server.Config
	var status helperStatus

	system.Info.Type = state.SelfHostedSystem
	system.Info.SelfHosted = &state.SystemInfoSelfHosted{
		Architecture: runtime.GOARCH,
	}

	statusBytes, err := exec.Command("/usr/bin/pganalyze-collector-helper", "status", config.DbDataDirectory).Output()
	if err != nil {
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error running system stats helper process: %s ", err))
		logger.PrintVerbose("Selfhosted/System: Could not run helper process: %s", err)
	} else {
		err = json.Unmarshal(statusBytes, &status)
		if err != nil {
			server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error reading system stats helper output: %s ", err))
			logger.PrintVerbose("Selfhosted/System: Could not unmarshal helper status: %s", err)
		}

		system.XlogUsedBytes = status.XlogUsedBytes
		system.Info.SelfHosted.DatabaseSystemIdentifier = status.SystemIdentifier
	}

	hostInfo, err := host.Info()
	if err != nil {
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error getting host information: %s ", err))
		logger.PrintVerbose("Selfhosted/System: Failed to get host information: %s", err)
	} else {
		system.Info.BootTime = time.Unix(int64(hostInfo.BootTime), 0)
		system.Info.SelfHosted.Hostname = hostInfo.Hostname
		system.Info.SelfHosted.OperatingSystem = hostInfo.OS
		system.Info.SelfHosted.Platform = hostInfo.Platform
		system.Info.SelfHosted.PlatformFamily = hostInfo.PlatformFamily
		system.Info.SelfHosted.PlatformVersion = hostInfo.PlatformVersion
		system.Info.SelfHosted.KernelVersion = hostInfo.KernelVersion

		if hostInfo.VirtualizationRole == "guest" {
			system.Info.SelfHosted.VirtualizationSystem = hostInfo.VirtualizationSystem
		}
	}

	loadAvg, err := load.Avg()
	if err != nil {
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error getting load average: %s ", err))
		logger.PrintVerbose("Selfhosted/System: Failed to get load average: %s", err)
	} else {
		system.Scheduler.Loadavg1min = loadAvg.Load1
		system.Scheduler.Loadavg5min = loadAvg.Load5
		system.Scheduler.Loadavg15min = loadAvg.Load15
	}

	memory, err := mem.VirtualMemory()
	if err != nil {
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error getting virtual memory stats: %s ", err))
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
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error getting swap stats: %s ", err))
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
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error getting CPU info: %s ", err))
		logger.PrintVerbose("Selfhosted/System: Failed to get CPU info: %s", err)
	} else if len(cpuInfos) > 0 {
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
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error getting CPU stats: %s ", err))
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
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error getting network stats: %s ", err))
		logger.PrintVerbose("Selfhosted/System: Failed to get network stats: %s", err)
	} else {
		system.NetworkStats = make(state.NetworkStatsMap)
		for _, netStat := range netStats {
			if (netStat.BytesRecv == 0 && netStat.BytesSent == 0) || netStat.Name == "lo" {
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
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error getting disk I/O stats: %s ", err))
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
				ReadsMerged:     disk.MergedReadCount,
				BytesRead:       disk.ReadBytes,
				ReadTimeMs:      disk.ReadTime,
				WritesCompleted: disk.WriteCount,
				WritesMerged:    disk.MergedWriteCount,
				BytesWritten:    disk.WriteBytes,
				WriteTimeMs:     disk.WriteTime,
				AvgQueueSize:    int32(disk.IopsInProgress),
				IoTime:          disk.IoTime,
			}
		}
	}

	// Remember disk components for software RAID
	fs, err := procfs.NewFS("/proc")
	if err != nil {
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error reading /proc: %s ", err))
		logger.PrintVerbose("Selfhosted/System: Could not access /proc: %s", err)
	} else {
		mdstats, err := fs.MDStat()
		if err != nil {
			server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error reading mdstat: %s ", err))
			logger.PrintVerbose("Selfhosted/System: Failed to get mdstat: %s", err)
		} else {
			for _, mdstat := range mdstats {
				mdDisk, exists := system.Disks[mdstat.Name]
				if exists {
					mdDisk.ComponentDisks = mdstat.Devices
					system.Disks[mdstat.Name] = mdDisk
				} else {
					system.Disks[mdstat.Name] = state.Disk{
						ComponentDisks: mdstat.Devices,
					}
				}
			}
		}
	}

	diskPartitions, err := disk.Partitions(true)
	if err != nil {
		server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error reading disk partitions: %s ", err))
		logger.PrintVerbose("Selfhosted/System: Failed to get disk partitions: %s", err)
	} else {
		system.DiskPartitions = make(state.DiskPartitionMap)
		for _, partition := range diskPartitions {
			// Linux partition types we can ignore
			if partition.Fstype == "devtmpfs" || partition.Fstype == "tmpfs" || partition.Fstype == "devpts" ||
				partition.Fstype == "fusectl" || partition.Fstype == "proc" || partition.Fstype == "squashfs" ||
				partition.Fstype == "securityfs" || partition.Fstype == "debugfs" || partition.Fstype == "sysfs" ||
				partition.Fstype == "pstore" || partition.Fstype == "mqueue" || partition.Fstype == "hugetlbfs" ||
				partition.Fstype == "cgroup" || partition.Fstype == "cgroup2" || partition.Fstype == "configfs" ||
				partition.Fstype == "fuse.lxcfs" {
				continue
			}

			// OSX partition types we can ignore
			if partition.Fstype == "autofs" || partition.Fstype == "devfs" {
				continue
			}

			diskUsage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				server.SelfCheckMarkSystemStatsError(fmt.Sprintf("error reading partition usage stats for %s: %s ", partition.Mountpoint, err))
				logger.PrintVerbose("Selfhosted/System: Failed to get disk partition usage stats for %s: %s", partition.Mountpoint, err)
			} else {
				var diskName string

				for name := range system.Disks {
					if (strings.HasPrefix(partition.Device, name) || strings.HasPrefix(partition.Device, "/dev/"+name)) && len(diskName) < len(name) {
						diskName = name
					}
				}

				if status.DataDirectory != "" && strings.HasPrefix(status.DataDirectory, partition.Mountpoint) && len(system.DataDirectoryPartition) < len(partition.Mountpoint) {
					system.DataDirectoryPartition = partition.Mountpoint
				}
				if status.XlogDirectory != "" && strings.HasPrefix(status.XlogDirectory, partition.Mountpoint) && len(system.XlogPartition) < len(partition.Mountpoint) {
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

	server.SelfCheckMarkSystemStatsOk()

	return
}
