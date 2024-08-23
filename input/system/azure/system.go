package azure

import (
	"context"
	"strings"

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
	} else if strings.ToLower(config.AzureResourceType) == "flexibleservers" {
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

	clientFactory, err := armpostgresqlflexibleservers.NewClientFactory(config.AzureSubscriptionID, credential, nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error making a client: %v", err)
		logger.PrintError("Azure/System: Encountered error making a client: %v\n", err)
		return
	}
	res, err := clientFactory.NewServersClient().Get(ctx, config.AzureResourceGroup, config.AzureDbServerName, nil)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting a server info: %v", err)
		logger.PrintError("Azure/System: Encountered error getting a server info: %v\n", err)
		return
	}

	logger.PrintInfo("server info: %+v", res)

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectSystemStats)

	return
}
