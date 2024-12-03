package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmosforpostgresql/armcosmosforpostgresql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v4"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState - Gets system information about an Azure instance
func GetSystemState(ctx context.Context, server *state.Server, logger *util.Logger) (system state.SystemState) {
	config := server.Config
	system.Info.Type = state.AzureDatabaseSystem
	if config.AzureSubscriptionID == "" {
		server.SelfTest.MarkCollectionAspectWarning(state.CollectionAspectSystemStats, "unable to collect system stats")
		server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Config value azure_subscription_id / AZURE_SUBSCRIPTION_ID is required to collect system stats.")
		return
	}

	credential, err := getAzureCredential(config)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting credential: %v", err)
		server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Make sure the managed identity is assigned to the collector VM.")
		logger.PrintError("Azure/System: Encountered error getting credential: %v\n", err)
		return
	}

	var resourceID string

	// Server info: Flexible Server
	clientFactory, err := armpostgresqlflexibleservers.NewClientFactory(config.AzureSubscriptionID, credential, nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error making a Flexible Server client: %v", err)
		logger.PrintError("Azure/System: Failed to make a Flexible Server client: %v\n", err)
		return
	}
	// Search a server from the list
	pager := clientFactory.NewServersClient().NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			// This likely raises errors with 403 AuthorizationFailed when the managed identity is not assigned to any role / any DB instances
			server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error advancing page of Flexible Server list: %v", err)
			server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Make sure the Monitoring Reader permission of the database is granted to the managed identity.")
			logger.PrintError("Azure/System: Failed to advance page of Flexible Server list: %v\n", err)
			return
		}
		for _, v := range page.Value {
			if v.ID != nil {
				rID, err := arm.ParseResourceID(*v.ID)
				if err != nil {
					logger.PrintError("Azure/System: Failed to parse a resource ID: %v\n", err)
					break
				}

				if config.AzureDbServerName == rID.Name {
					customWindowEnabled := util.StringCustomTypePtrToString(v.Properties.MaintenanceWindow.CustomWindow) == "Enabled"
					if v.Properties.SourceServerResourceID != nil {
						sourceID, err := arm.ParseResourceID(*v.Properties.SourceServerResourceID)
						if err == nil {
							system.Info.ClusterID = fmt.Sprintf("%s/%s", sourceID.ResourceGroupName, sourceID.Name)
						}
					} else {
						system.Info.ClusterID = fmt.Sprintf("%s/%s", rID.ResourceGroupName, rID.Name)
					}
					system.Info.Azure = &state.SystemInfoAzure{
						Location:                util.StringPtrToString(v.Location),
						CreatedAt:               util.TimePtrToTime(v.SystemData.CreatedAt),
						State:                   util.StringCustomTypePtrToString(v.Properties.State),
						SubscriptionID:          config.AzureSubscriptionID,
						ResourceGroup:           rID.ResourceGroupName,
						ResourceType:            rID.ResourceType.Type,
						ResourceName:            rID.Name,
						MaintenanceCustomWindow: customWindowEnabled,
						MaintenanceDayOfWeek:    util.Int32PtrToInt(v.Properties.MaintenanceWindow.DayOfWeek),
						MaintenanceStartHour:    util.Int32PtrToInt(v.Properties.MaintenanceWindow.StartHour),
						MaintenanceStartMinute:  util.Int32PtrToInt(v.Properties.MaintenanceWindow.StartMinute),
						SKUName:                 util.StringPtrToString(v.SKU.Name),
						AvailabilityZone:        util.StringPtrToString(v.Properties.AvailabilityZone),
						StorageGB:               util.Int32PtrToInt(v.Properties.Storage.StorageSizeGB),
						HighAvailabilityMode:    util.StringCustomTypePtrToString(v.Properties.HighAvailability.Mode),
						HighAvailabilityState:   util.StringCustomTypePtrToString(v.Properties.HighAvailability.State),
						ReplicationRole:         util.StringCustomTypePtrToString(v.Properties.ReplicationRole),
					}
					tags := make(map[string]string)
					for key, value := range v.Tags {
						tags[key] = util.StringPtrToString(value)
					}
					system.Info.ResourceTags = tags
					resourceID = *v.ID
					break
				}
			}
		}
	}

	// Server info: Cosmos DB (when server is not found within Flexible Server)
	if resourceID == "" {
		clientFactory, err := armcosmosforpostgresql.NewClientFactory(config.AzureSubscriptionID, credential, nil)
		if err != nil {
			server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error making a Cosmos DB client: %v", err)
			logger.PrintError("Azure/System: Failed to make a Cosmos DB client: %v\n", err)
			return
		}
		// Search a server from the list
		pager := clientFactory.NewClustersClient().NewListPager(nil)
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				logger.PrintError("Azure/System: Failed to advance page of Cosmos DB cluster list: %v\n", err)
				break
			}
			for _, v := range page.Value {
				if v.ID != nil {
					rID, err := arm.ParseResourceID(*v.ID)
					if err != nil {
						logger.PrintError("Azure/System: Failed to parse a resource ID: %v\n", err)
						break
					}

					if config.AzureDbServerName == rID.Name {
						customWindowEnabled := util.StringCustomTypePtrToString(v.Properties.MaintenanceWindow.CustomWindow) == "Enabled"
						system.Info.ClusterID = fmt.Sprintf("%s/%s", rID.ResourceGroupName, rID.Name)
						system.Info.Azure = &state.SystemInfoAzure{
							Location:                 util.StringPtrToString(v.Location),
							CreatedAt:                util.TimePtrToTime(v.SystemData.CreatedAt),
							State:                    util.StringCustomTypePtrToString(v.Properties.State),
							SubscriptionID:           config.AzureSubscriptionID,
							ResourceGroup:            rID.ResourceGroupName,
							ResourceType:             rID.ResourceType.Type,
							ResourceName:             rID.Name,
							MaintenanceCustomWindow:  customWindowEnabled,
							MaintenanceDayOfWeek:     util.Int32PtrToInt(v.Properties.MaintenanceWindow.DayOfWeek),
							MaintenanceStartHour:     util.Int32PtrToInt(v.Properties.MaintenanceWindow.StartHour),
							MaintenanceStartMinute:   util.Int32PtrToInt(v.Properties.MaintenanceWindow.StartMinute),
							CitusVersion:             util.StringPtrToString(v.Properties.CitusVersion),
							HighAvailabilityEnabled:  util.BoolPtrToBool(v.Properties.EnableHa),
							CoordinatorStorageMB:     util.Int32PtrToInt(v.Properties.CoordinatorStorageQuotaInMb),
							NodeStorageMB:            util.Int32PtrToInt(v.Properties.NodeStorageQuotaInMb),
							CoordinatorVCores:        util.Int32PtrToInt(v.Properties.CoordinatorVCores),
							NodeVCores:               util.Int32PtrToInt(v.Properties.NodeVCores),
							CoordinatorServerEdition: util.StringPtrToString(v.Properties.CoordinatorServerEdition),
							NodeServerEdition:        util.StringPtrToString(v.Properties.NodeServerEdition),
							NodeCount:                util.Int32PtrToInt(v.Properties.NodeCount),
						}
						tags := make(map[string]string)
						for key, value := range v.Tags {
							tags[key] = util.StringPtrToString(value)
						}
						system.Info.ResourceTags = tags
						resourceID = *v.ID
						break
					}
				}
			}
		}
	}

	if resourceID == "" {
		// This is reached when the managed identity is assigned to _some_ databases but not the one that we want to get the info
		server.SelfTest.MarkCollectionAspectWarning(state.CollectionAspectSystemStats, "unable to find the database server info")
		server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Make sure the Monitoring Reader permission of the database is granted to the managed identity.")
		return
	}

	// Server metrics
	client, err := azquery.NewMetricsClient(credential, nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error making a metrics client: %v", err)
		logger.PrintError("Azure/System: Failed to make a metrics client: %v\n", err)
		return
	}

	// Query metrics data with 1 min interval, for the last 1 min (should return 1 value)
	metricNames := "cpu_percent,memory_percent,network_bytes_egress,network_bytes_ingress,storage_used"
	if strings.ToLower(system.Info.Azure.ResourceType) == "flexibleservers" {
		// metrics only available with Flexible Server
		metricNames += ",txlogs_storage_used,read_iops,write_iops,disk_queue_depth,read_throughput,write_throughput"
	}
	option := &azquery.MetricsClientQueryResourceOptions{
		MetricNames: to.Ptr(metricNames),
		Aggregation: to.SliceOfPtrs(azquery.AggregationTypeAverage),
		Interval:    to.Ptr("PT1M"),
		Timespan:    to.Ptr(azquery.TimeInterval("PT1M")),
	}

	metricsRes, err := client.QueryResource(ctx, resourceID, option)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting server metrics: %v", err)
		logger.PrintError("Azure/System: Encountered error getting server metrics: %v\n", err)
		server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Make sure the Monitoring Reader permission of the database is granted to the managed identity.")
		return
	}

	system.CPUStats = make(state.CPUStatisticMap)
	diffedNetworkStats := &state.DiffedNetworkStats{}
	diffedDiskStats := &state.DiffedDiskStats{}
	for _, metric := range metricsRes.Value {
		// Should be only one data as 1 min time span with 1 min interval is selected, so getting first metric is good
		metricValue := getFirstMetricValue(metric)
		if metricValue == nil || metricValue.Average == nil {
			continue
		}
		switch *metric.Name.Value {
		case "cpu_percent":
			system.CPUStats["all"] = state.CPUStatistic{
				DiffedOnInput: true,
				DiffedValues: &state.DiffedSystemCPUStats{
					UserPercent: *metricValue.Average,
				},
			}
		case "memory_percent":
			// Currently, we can only retrieve memory percent
			// Since total memory size is not listed in the server info,
			// we are unable to pass this value to pganalyze at the moment
		case "txlogs_storage_used":
			system.XlogUsedBytes = uint64(*metricValue.Average)
		case "network_bytes_egress":
			// value is total of 1min
			diffedNetworkStats.TransmitThroughputBytesPerSecond = uint64(*metricValue.Average / 60)
		case "network_bytes_ingress":
			// value is total of 1min
			diffedNetworkStats.ReceiveThroughputBytesPerSecond = uint64(*metricValue.Average / 60)
		case "read_iops":
			diffedDiskStats.ReadOperationsPerSecond = *metricValue.Average
		case "write_iops":
			diffedDiskStats.WriteOperationsPerSecond = *metricValue.Average
		case "disk_queue_depth":
			diffedDiskStats.AvgQueueSize = int32(*metricValue.Average)
		case "read_throughput":
			diffedDiskStats.BytesReadPerSecond = *metricValue.Average
		case "write_throughput":
			diffedDiskStats.BytesWrittenPerSecond = *metricValue.Average
		case "storage_used":
			if system.Info.Azure.StorageGB != 0 {
				// Flexible Server
				totalGB := uint64(system.Info.Azure.StorageGB)
				system.DiskPartitions = make(state.DiskPartitionMap)
				system.DiskPartitions["/"] = state.DiskPartition{
					DiskName:   "default",
					UsedBytes:  uint64(*metricValue.Average),
					TotalBytes: totalGB * 1024 * 1024 * 1024,
				}
			} else if system.Info.Azure.CoordinatorStorageMB != 0 {
				// Cosmos DB
				totalMB := uint64(system.Info.Azure.CoordinatorStorageMB)
				system.DiskPartitions = make(state.DiskPartitionMap)
				system.DiskPartitions["/"] = state.DiskPartition{
					DiskName:   "default",
					UsedBytes:  uint64(*metricValue.Average),
					TotalBytes: totalMB * 1024 * 1024,
				}
			}
		}
	}
	system.NetworkStats = make(state.NetworkStatsMap)
	system.NetworkStats["default"] = state.NetworkStats{
		DiffedOnInput: true,
		DiffedValues:  diffedNetworkStats,
	}
	system.Disks = make(state.DiskMap)
	system.Disks["default"] = state.Disk{}
	if strings.ToLower(system.Info.Azure.ResourceType) == "flexibleservers" {
		// DiskStats is only available with Flexible Server
		system.DiskStats = make(state.DiskStatsMap)
		system.DiskStats["default"] = state.DiskStats{
			DiffedOnInput: true,
			DiffedValues:  diffedDiskStats,
		}
	}

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectSystemStats)

	return
}

// getFirstMetricValue gets the first data from the time series metric and returns the value
func getFirstMetricValue(metric *azquery.Metric) (metricValue *azquery.MetricValue) {
        if len(metric.TimeSeries) == 0 || len(metric.TimeSeries[0].timeSeriesElement.Data) == 0 {
                return
        }
        return metric.TimeSeries[0].timeSeriesElement.Data[0]
}
