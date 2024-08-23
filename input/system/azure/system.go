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
	option := &azquery.MetricsClientQueryResourceOptions{MetricNames: to.Ptr("cpu_percent,database_size_bytes,memory_percent,storage_used")}

	metricsRes, err := client.QueryResource(ctx, config.AzureResourceID, option)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting server metrics: %v", err)
		logger.PrintError("Azure/System: Encountered error getting server metrics: %v\n", err)
		server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Make sure the Monitoring Reader permission of the database is granted to the managed identity.")
		return
	}
	for _, metric := range metricsRes.Value {
		logger.PrintInfo("Metrics name: %s", *metric.Name.Value)
		for _, timeSeriesElement := range metric.TimeSeries {
			for _, metricValue := range timeSeriesElement.Data {
				logger.PrintInfo("Metrics time: %v", metricValue.TimeStamp)
			}
		}
	}

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectSystemStats)

	return
}
