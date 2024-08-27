package azure

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v4"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState - Gets system information about an Azure instance
func GetSystemState(ctx context.Context, server *state.Server, logger *util.Logger) (system state.SystemState) {
	config := server.Config
	system.Info.Type = state.AzureDatabaseSystem
	if config.AzureResourceID == "" {
		server.SelfTest.MarkCollectionAspectWarning(state.CollectionAspectSystemStats, "unable to collect system stats")
		server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Config value azure_resource_id is required to collect system stats.")
		return
	} else if strings.ToLower(config.AzureResourceType) != "flexibleservers" {
		server.SelfTest.MarkCollectionAspectWarning(state.CollectionAspectSystemStats, "unable to collect system stats")
		server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "System stats collection is only supported for Flexible Server.")
		return
	}

	credential, err := getAzureCredential(config)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting credential: %v", err)
		logger.PrintError("Azure/System: Encountered error getting credential: %v\n", err)
		return
	}

	// Server info
	clientFactory, err := armpostgresqlflexibleservers.NewClientFactory(config.AzureSubscriptionID, credential, nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error making a client: %v", err)
		logger.PrintError("Azure/System: Failed to make a factory client: %v\n", err)
		return
	}
	serverRes, err := clientFactory.NewServersClient().Get(ctx, config.AzureResourceGroup, config.AzureDbServerName, nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting a server info: %v", err)
		server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Make sure the Reader permission of the database is granted to the managed identity.")
		logger.PrintError("Azure/System: Encountered error getting a server info: %v\n", err)
		return
	}

	system.Info.Azure = &state.SystemInfoAzure{
		Location:              util.StringPtrToString(serverRes.Location),
		CreatedAt:             util.TimePtrToTime(serverRes.SystemData.CreatedAt),
		State:                 util.StringCustomTypePtrToString(serverRes.Properties.State),
		AvailabilityZone:      util.StringPtrToString(serverRes.Properties.AvailabilityZone),
		ResourceGroup:         config.AzureResourceGroup,
		StorageGB:             util.Int32PtrToInt(serverRes.Properties.Storage.StorageSizeGB),
		HighAvailabilityMode:  util.StringCustomTypePtrToString(serverRes.Properties.HighAvailability.Mode),
		HighAvailabilityState: util.StringCustomTypePtrToString(serverRes.Properties.HighAvailability.State),
		ReplicationRole:       util.StringCustomTypePtrToString(serverRes.Properties.ReplicationRole),
	}

	// Server metrics
	client, err := azquery.NewMetricsClient(credential, nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error making a metrics client: %v", err)
		logger.PrintError("Azure/System: Failed to make a metrics client: %v\n", err)
		return
	}

	// Query metrics data with 1 min interval, for the last 1 min (should return 1 value)
	metricNames := "cpu_percent,memory_percent,txlogs_storage_used,network_bytes_egress,network_bytes_ingress," +
		"read_iops,write_iops,disk_queue_depth,read_throughput,write_throughput,storage_used"
	option := &azquery.MetricsClientQueryResourceOptions{
		MetricNames: to.Ptr(metricNames),
		Aggregation: to.SliceOfPtrs(azquery.AggregationTypeAverage),
		Interval:    to.Ptr("PT1M"),
		Timespan:    to.Ptr(azquery.TimeInterval("PT1M")),
	}

	metricsRes, err := client.QueryResource(ctx, config.AzureResourceID, option)
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
		if *metric.Name.Value == "cpu_percent" {
			// Should be only one data as 1 min time span with 1 min interval is selected
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						system.CPUStats["all"] = state.CPUStatistic{
							DiffedOnInput: true,
							DiffedValues: &state.DiffedSystemCPUStats{
								UserPercent: *metricValue.Average,
							},
						}
					}
				}
			}
		} else if *metric.Name.Value == "memory_percent" {
			// Currently, we can only retrieve memory percent
			// Since total memory size is not listed in the server info,
			// we are unable to pass this value to pganalyze at the moment
		} else if *metric.Name.Value == "txlogs_storage_used" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						system.XlogUsedBytes = uint64(*metricValue.Average)
					}
				}
			}
		} else if *metric.Name.Value == "network_bytes_egress" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						// value is total of 1min
						diffedNetworkStats.TransmitThroughputBytesPerSecond = uint64(*metricValue.Average / 60)
					}
				}
			}
		} else if *metric.Name.Value == "network_bytes_ingress" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						// value is total of 1min
						diffedNetworkStats.ReceiveThroughputBytesPerSecond = uint64(*metricValue.Average / 60)
					}
				}
			}
		} else if *metric.Name.Value == "read_iops" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						diffedDiskStats.ReadOperationsPerSecond = *metricValue.Average
					}
				}
			}
		} else if *metric.Name.Value == "write_iops" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						diffedDiskStats.WriteOperationsPerSecond = *metricValue.Average
					}
				}
			}
		} else if *metric.Name.Value == "disk_queue_depth" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						diffedDiskStats.AvgQueueSize = int32(*metricValue.Average)
					}
				}
			}
		} else if *metric.Name.Value == "read_throughput" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						diffedDiskStats.BytesReadPerSecond = *metricValue.Average
					}
				}
			}
		} else if *metric.Name.Value == "write_throughput" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil {
						diffedDiskStats.BytesWrittenPerSecond = *metricValue.Average
					}
				}
			}
		} else if *metric.Name.Value == "storage_used" {
			for _, timeSeriesElement := range metric.TimeSeries {
				for _, metricValue := range timeSeriesElement.Data {
					if metricValue.Average != nil && serverRes.Properties.Storage.StorageSizeGB != nil {
						totalGB := uint64(*serverRes.Properties.Storage.StorageSizeGB)
						system.DiskPartitions = make(state.DiskPartitionMap)
						system.DiskPartitions["/"] = state.DiskPartition{
							DiskName:   "default",
							UsedBytes:  uint64(*metricValue.Average),
							TotalBytes: totalGB * 1024 * 1024 * 1024,
						}
					}
				}
			}
		}
	}
	system.NetworkStats = make(state.NetworkStatsMap)
	system.NetworkStats["default"] = state.NetworkStats{
		DiffedOnInput: true,
		DiffedValues:  diffedNetworkStats,
	}
	system.DiskStats["default"] = state.DiskStats{
		DiffedOnInput: true,
		DiffedValues:  diffedDiskStats,
	}

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectSystemStats)

	return
}
