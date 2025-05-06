package crunchy_bridge

import (
	"context"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState - Gets system information about a Crunchy Bridge instance
func GetSystemState(ctx context.Context, server *state.Server, logger *util.Logger) (system state.SystemState) {
	config := server.Config
	system.Info.Type = state.CrunchyBridgeSystem

	// When API key is provided, use API to obtain cluster info and system metrics
	if config.CrunchyBridgeAPIKey == "" {
		return
	}
	client := Client{Client: *config.HTTPClientWithRetry, BaseURL: apiBaseURL, BearerToken: config.CrunchyBridgeAPIKey, ClusterID: config.CrunchyBridgeClusterID}

	clusterInfo, err := client.GetClusterInfo(ctx)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting cluster info: %s", err)
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster info %v\n", err)
		return
	}
	if clusterInfo.ParentID.Valid {
		system.Info.ClusterID = clusterInfo.ParentID.String
	} else {
		system.Info.ClusterID = client.ClusterID
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

	cpuMetrics, err := client.GetCPUMetrics(ctx)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting cluster CPU metrics: %s", err)
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster CPU metrics %v\n", err)
		return
	}
	system.CPUStats["all"] = state.CPUStatistic{
		DiffedOnInput: true,
		DiffedValues: &state.DiffedSystemCPUStats{
			IowaitPercent: cpuMetrics.Iowait,
			SystemPercent: cpuMetrics.System,
			UserPercent:   cpuMetrics.User,
			StealPercent:  cpuMetrics.Steal,
		},
	}

	loadAverageMetrics, err := client.GetLoadAverageMetrics(ctx)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting cluster Load Average metrics: %s", err)
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster Load Average metrics %v\n", err)
		return
	}
	system.Scheduler.Loadavg1min = loadAverageMetrics.One

	memoryMetrics, err := client.GetMemoryMetrics(ctx)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting cluster memory metrics: %s", err)
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster memory metrics %v\n", err)
		return
	}
	totalMemoryInBytes := clusterInfo.Memory * 1024 * 1024 * 1024
	system.Memory.TotalBytes = uint64(totalMemoryInBytes)
	// ApplicationBytes is not the best way for representing "used bytes",
	// but in the UI side, we use this as "process" memory if this value exits
	// which would be the closest to the used bytes
	system.Memory.ApplicationBytes = uint64(float64(totalMemoryInBytes) * memoryMetrics.MemoryUsedPct)
	system.Memory.SwapUsedBytes = uint64(float64(totalMemoryInBytes) * memoryMetrics.SwapUsedPct)

	iopsMetrics, err := client.GetIOPSMetrics(ctx)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting cluster IOPS metrics: %s", err)
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

	diskUsageMetrics, err := client.GetDiskUsageMetrics(ctx)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting cluster disk usage metrics: %s", err)
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster disk usage metrics %v\n", err)
		return
	}

	system.DataDirectoryPartition = "/"
	system.DiskPartitions = make(state.DiskPartitionMap)
	// Manually specify the disk name to "md0" as that's the main disk
	system.DiskPartitions["/"] = state.DiskPartition{
		DiskName:      "md0",
		PartitionName: "md0",
		UsedBytes:     diskUsageMetrics.DatabaseSize,
		TotalBytes:    uint64(clusterInfo.Storage) * 1024 * 1024 * 1024,
	}
	system.XlogUsedBytes = diskUsageMetrics.WalSize

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectSystemStats)

	return
}
