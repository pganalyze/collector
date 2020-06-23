package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	"github.com/Azure/azure-amqp-common-go/aad"
	"github.com/Azure/azure-amqp-common-go/persist"
	eventhubs "github.com/Azure/azure-event-hubs-go"
	"github.com/Azure/go-autorest/autorest/azure"
)

var connectionReceivedRegexp = regexp.MustCompile(`^(connection received: host=[^ ]+( port=\w+)?) pid=\d+`)
var connectionAuthorizedRegexp = regexp.MustCompile(`^(connection authorized: user=\w+)(database=\w+)`)
var checkpointCompleteRegexp = regexp.MustCompile(`^(checkpoint complete) \(\d+\)(:)`)

func setupEventHubReceiver(ctx context.Context, wg *sync.WaitGroup, logger *util.Logger, config config.ServerConfig, azureLogStream chan AzurePostgresLogRecord) error {
	provider, err := aad.NewJWTProvider(func(c *aad.TokenProviderConfiguration) error {
		c.TenantID = config.AzureADTenantID
		c.ClientID = config.AzureADClientID
		c.ClientSecret = config.AzureADClientSecret
		c.CertificatePath = config.AzureADCertificatePath
		c.CertificatePassword = config.AzureADCertificatePassword
		c.Env = &azure.PublicCloud
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to configure Azure AD JWT provider: %s", err)
	}

	hub, err := eventhubs.NewHub(config.AzureEventhubNamespace, config.AzureEventhubName, provider)
	if err != nil {
		return fmt.Errorf("failed to configure Event Hub: %s", err)
	}

	info, err := hub.GetRuntimeInformation(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to the Event Hub management node: %s", err)
	}

	handler := func(ctx context.Context, event *eventhubs.Event) error {
		var eventData AzureEventHubData
		err := json.Unmarshal(event.Data, &eventData)
		if err != nil {
			logger.PrintWarning("Error parsing Azure Event Hub event: %s", err)
		}
		for _, record := range eventData.Records {
			if record.Category == "PostgreSQLLogs" && record.OperationName == "LogEvent" {
				// Adjust Azure-modified log messages to be standard Postgres log messages
				if strings.HasPrefix(record.Properties.Message, "connection received:") {
					record.Properties.Message = connectionReceivedRegexp.ReplaceAllString(record.Properties.Message, "$1")
				}
				if strings.HasPrefix(record.Properties.Message, "connection authorized:") {
					record.Properties.Message = connectionAuthorizedRegexp.ReplaceAllString(record.Properties.Message, "$1 $2")
				}
				if strings.HasPrefix(record.Properties.Message, "checkpoint complete") {
					record.Properties.Message = checkpointCompleteRegexp.ReplaceAllString(record.Properties.Message, "$1$2")
				}

				azureLogStream <- record

				// DETAIL messages are handled a bit weird here - for now we'll just fake a separate log message
				// to get them through. Note that other secondary log lines (CONTEXT, STATEMENT, etc) are missing
				// from the log stream.
				if record.Properties.Detail != "" {
					azureLogStream <- AzurePostgresLogRecord{
						LogicalServerName: record.LogicalServerName,
						SubscriptionID:    record.SubscriptionID,
						ResourceGroup:     record.ResourceGroup,
						Time:              record.Time,
						ResourceID:        record.ResourceID,
						Category:          record.Category,
						OperationName:     record.OperationName,
						Properties: AzurePostgresLogMessage{
							Prefix:       record.Properties.Prefix,
							Message:      record.Properties.Detail, // This is the important difference from the main message
							Detail:       "",
							ErrorLevel:   "DETAIL",
							Domain:       record.Properties.Domain,
							SchemaName:   record.Properties.SchemaName,
							TableName:    record.Properties.TableName,
							ColumnName:   record.Properties.ColumnName,
							DatatypeName: record.Properties.DatatypeName,
						},
					}
				}
			}
		}
		return nil
	}

	logger.PrintVerbose("Initializing Azure Event Hub handler")

	for _, partitionID := range info.PartitionIDs {
		_, err := hub.Receive(
			ctx,
			partitionID,
			handler,
			eventhubs.ReceiveWithStartingOffset(persist.StartOfStream),
		)
		if err != nil {
			return fmt.Errorf("failed to setup Azure Event Hub receiver for partition ID %s: %s", partitionID, err)
		}
	}

	return nil
}

func SetupLogSubscriber(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []state.Server, azureLogStream chan AzurePostgresLogRecord) error {
	// This map is used to avoid duplicate receivers to the same Azure Event Hub
	eventHubReceivers := make(map[string]bool)

	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)
		if server.Config.AzureEventhubNamespace != "" && server.Config.AzureEventhubName != "" {
			if _, ok := eventHubReceivers[server.Config.AzureEventhubNamespace+"/"+server.Config.AzureEventhubName]; ok {
				continue
			}
			err := setupEventHubReceiver(ctx, wg, prefixedLogger, server.Config, azureLogStream)
			if err != nil {
				if globalCollectionOpts.TestRun {
					return err
				}

				prefixedLogger.PrintWarning("Skipping logs, could not setup log subscriber: %s", err)
				continue
			}

			eventHubReceivers[server.Config.AzureEventhubNamespace+"/"+server.Config.AzureEventhubName] = true
		}
	}

	return nil
}
