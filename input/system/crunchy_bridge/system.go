package crunchy_bridge

import (
	"fmt"
	"time"

	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState - Gets system information about a Crunchy Bridge instance
func GetSystemState(server *state.Server, logger *util.Logger) (system state.SystemState) {
	config := server.Config
	// With Crunchy Bridge, we are assuming that the collector is deployed on Container Apps,
	// which run directly on the database server. Most of the metrics can be obtained
	// using the same way as a self hosted server, since the container receives a bind mount
	// of /proc and /sys from the host. Note this excludes disk usage metrics, which we instead
	// get from the API.
	system = selfhosted.GetSystemState(server, logger)
	system.Info.Type = state.CrunchyBridgeSystem

	// When API key is provided, use API to obtain extra info including disk usage metrics
	if config.CrunchyBridgeAPIKey == "" {
		return
	}
	client := Client{Client: *config.HTTPClientWithRetry, BaseURL: apiBaseURL, BearerToken: config.CrunchyBridgeAPIKey, ClusterID: config.CrunchyBridgeClusterID}

	clusterInfo, err := client.GetClusterInfo()
	if err != nil {
		server.SelfTestMarkSystemStatsError(fmt.Sprintf("error getting cluster info: %s\n", err))
		logger.PrintError("CrunchyBridge/System: Encountered error when getting cluster info %v\n", err)
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
		server.SelfTestMarkSystemStatsError(fmt.Sprintf("error getting cluster disk usage metrics: %s\n", err))
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

	server.SelfTestMarkSystemStatsOk()

	return
}
