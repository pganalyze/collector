package crunchy_bridge

import (
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState - Gets system information about a Crunchy Bridge instance
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
	system.Info.Type = state.CrunchyBridgeSystem
	if config.CrunchyBridgeAPIKey == "" {
		return
	}
	client := Client{Client: *config.HTTPClientWithRetry, BaseURL: apiBaseURL, BearerToken: config.CrunchyBridgeAPIKey, ClusterID: config.CrunchyBridgeClusterID}

	clusterInfo, err := client.GetClusterInfo()
	if err != nil {
		logger.PrintError("CrunchyBridge/System: Encountered error when getting a cluster info %v\n", err)
		return
	}
	system.Info.CrunchyBridge = &state.SystemInfoCrunchyBridge{
		ClusterName: clusterInfo.Name,
		PlanID:      clusterInfo.PlanID,
		ProviderID:  clusterInfo.ProviderID,
		RegionID:    clusterInfo.RegionID,
		CPUUnits:    clusterInfo.CPU,
		StorageGB:   clusterInfo.Storage,
		MemoryGB:    clusterInfo.Memory,
	}
	if parsedCreatedAt, err := time.Parse(time.RFC3339, clusterInfo.CreatedAt); err != nil {
		system.Info.CrunchyBridge.CreatedAt = parsedCreatedAt
	}

	cpuMetrics, err := client.GetCPUMetrics()
	if err != nil {
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster CPU metrics %v\n", err)
		return
	}

	system.CPUStats = make(state.CPUStatisticMap)
	system.CPUStats["all"] = state.CPUStatistic{
		DiffedOnInput: true,
		DiffedValues: &state.DiffedSystemCPUStats{
			IowaitPercent: cpuMetrics.Iowait,
			SystemPercent: cpuMetrics.System,
			UserPercent:   cpuMetrics.User,
			StealPercent:  cpuMetrics.Steal,
		},
	}

	iopsMetrics, err := client.GetIOPSMetrics()
	if err != nil {
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster IOPS metrics %v\n", err)
		return
	}

	system.Disks = make(state.DiskMap)
	system.Disks["default"] = state.Disk{}

	system.DiskStats = make(state.DiskStatsMap)
	system.DiskStats["default"] = state.DiskStats{
		DiffedOnInput: true,
		DiffedValues: &state.DiffedDiskStats{
			ReadOperationsPerSecond:  iopsMetrics.Reads,
			WriteOperationsPerSecond: iopsMetrics.Writes,
		},
	}

	loadAverageMetrics, err := client.GetLoadAverageMetrics()
	if err != nil {
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster load average metrics %v\n", err)
		return
	}

	system.Scheduler.Loadavg1min = loadAverageMetrics.One
	system.CPUInfo.SocketCount = 1
	system.CPUInfo.LogicalCoreCount = clusterInfo.CPU
	system.CPUInfo.PhysicalCoreCount = clusterInfo.CPU

	diskUsageMetrics, err := client.GetDiskUsageMetrics()
	if err != nil {
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster disk usage metrics %v\n", err)
		return
	}

	system.DataDirectoryPartition = "/"
	system.DiskPartitions = make(state.DiskPartitionMap)
	system.DiskPartitions["/"] = state.DiskPartition{
		DiskName:      "default",
		PartitionName: "default",
		UsedBytes:     diskUsageMetrics.DatabaseSize,
		TotalBytes:    uint64(clusterInfo.Storage) * 1024 * 1024 * 1024,
	}

	return
}
