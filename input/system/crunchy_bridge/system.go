package crunchy_bridge

import (
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState - Gets system information about a Crunchy Bridge instance
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
	// With Crunchy Bridge, we are assuming that the collector is deployed on the Container Apps
	// Most of the metrics can be obtained using the same way as the selfhosted
	// (as the collector runs on the database server), except disk usage metrics
	system = selfhosted.GetSystemState(config, logger)
	system.Info.Type = state.CrunchyBridgeSystem

	// When API key is provided, use API to obtain extra info including disk usage metrics
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

	diskUsageMetrics, err := client.GetDiskUsageMetrics()
	if err != nil {
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster disk usage metrics %v\n", err)
		return
	}

	system.DataDirectoryPartition = "/"
	system.DiskPartitions = make(state.DiskPartitionMap)
	system.DiskPartitions["/"] = state.DiskPartition{
		DiskName:      "data",
		PartitionName: "data",
		UsedBytes:     diskUsageMetrics.DatabaseSize,
		TotalBytes:    uint64(clusterInfo.Storage) * 1024 * 1024 * 1024,
	}
	system.XlogUsedBytes = diskUsageMetrics.WalSize

	return
}
