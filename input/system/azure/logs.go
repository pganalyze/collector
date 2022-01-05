package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"

	"github.com/Azure/azure-amqp-common-go/v3/aad"
	eventhubs "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/persist"
	"github.com/Azure/go-autorest/autorest/azure"
)

type AzurePostgresLogMessage struct {
	Prefix       string `json:"prefix"`
	Message      string `json:"message"`
	Detail       string `json:"detail"`
	ErrorLevel   string `json:"errorLevel"`
	Domain       string `json:"domain"`
	SchemaName   string `json:"schemaName"`
	TableName    string `json:"tableName"`
	ColumnName   string `json:"columnName"`
	DatatypeName string `json:"datatypeName"`
}

type AzurePostgresLogRecord struct {
	LogicalServerName string                  `json:"LogicalServerName"`
	SubscriptionID    string                  `json:"SubscriptionId"`
	ResourceGroup     string                  `json:"ResourceGroup"`
	Time              string                  `json:"time"`
	ResourceID        string                  `json:"resourceId"`
	Category          string                  `json:"category"`
	OperationName     string                  `json:"operationName"`
	Properties        AzurePostgresLogMessage `json:"properties"`
}

type AzureEventHubData struct {
	Records []AzurePostgresLogRecord `json:"records"`
}

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

func SetupLogSubscriber(ctx context.Context, wg *sync.WaitGroup, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) error {
	azureLogStream := make(chan AzurePostgresLogRecord, state.LogStreamBufferLen)
	setupLogTransformer(ctx, wg, servers, azureLogStream, parsedLogStream, globalCollectionOpts, logger)

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

func setupLogTransformer(ctx context.Context, wg *sync.WaitGroup, servers []*state.Server, in <-chan AzurePostgresLogRecord, out chan state.ParsedLogStreamItem, globalCollectionOpts state.CollectionOpts, logger *util.Logger) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Only ingest log lines that were written in the last minute before startup
		// (Azure Event Hub will return older log lines as well)
		linesNewerThan := time.Now().Add(-1 * time.Minute)

		for {
			select {
			case <-ctx.Done():
				return
			case in, ok := <-in:
				if !ok {
					return
				}

				logLineContent := fmt.Sprintf("%s%s:  %s", in.Properties.Prefix, in.Properties.ErrorLevel, in.Properties.Message)
				logLine, ok := logs.ParseLogLineWithPrefix("", logLineContent)
				if !ok {
					logger.PrintError("Can't parse log line: \"%s\"", logLineContent)
					continue
				}
				logLine.CollectedAt = time.Now()
				logLine.UUID = uuid.NewV4()

				// Ignore loglines which are outside our time window (except in test runs)
				if !logLine.OccurredAt.IsZero() && logLine.OccurredAt.Before(linesNewerThan) && !globalCollectionOpts.TestRun {
					continue
				}

				for _, server := range servers {
					if in.LogicalServerName == server.Config.AzureDbServerName {
						out <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
					}
				}

			}
		}
	}()
}
