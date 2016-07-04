package selfhosted

import (
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

// GetSystemState - Gets system information about a self-hosted (physical/virtual) system
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
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
				MeasuredAsSeconds: true,
				UserSeconds:       cpuStat.User,
				SystemSeconds:     cpuStat.System,
				IdleSeconds:       cpuStat.Idle,
				NiceSeconds:       cpuStat.Nice,
				IowaitSeconds:     cpuStat.Iowait,
				IrqSeconds:        cpuStat.Irq,
				SoftIrqSeconds:    cpuStat.Softirq,
				StealSeconds:      cpuStat.Steal,
				GuestSeconds:      cpuStat.Guest,
				GuestNiceSeconds:  cpuStat.GuestNice,
			}
		}
	}

	netStats, err := net.IOCounters(true)
	if err != nil {
		logger.PrintVerbose("Selfhosted/System: Failed to get network stats: %s", err)
	} else {
		system.NetworkStats = make(state.NetworkStatsMap)
		for _, netStat := range netStats {
			system.NetworkStats[netStat.Name] = state.NetworkStats{
				ReceiveThroughputBytes:  netStat.BytesRecv,
				TransmitThroughputBytes: netStat.BytesSent,
			}
		}
	}

	return
}
