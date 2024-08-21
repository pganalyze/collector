package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
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

func (record *AzurePostgresLogRecord) IsSingleServer() bool {
	resourceParts := strings.Split(record.ResourceID, "/")
	// ResourceID's second to last element is a resource type
	return strings.ToLower(resourceParts[len(resourceParts)-2]) == "servers"
}

func (record *AzurePostgresLogRecord) IsCosmosDB() bool {
	resourceParts := strings.Split(record.ResourceID, "/")
	return strings.ToLower(resourceParts[len(resourceParts)-2]) == "servergroupsv2"
}

type AzureEventHubData struct {
	Records []AzurePostgresLogRecord `json:"records"`
}

var connectionReceivedRegexp = regexp.MustCompile(`^(connection received: host=[^ ]+( port=\w+)?) pid=\d+`)
var connectionAuthorizedRegexp = regexp.MustCompile(`^(connection authorized: user=\w+)(database=\w+)`)
var checkpointCompleteRegexp = regexp.MustCompile(`^(checkpoint complete) \(\d+\)(:)`)

func getEventHubConsumerClient(config config.ServerConfig) (*azeventhubs.ConsumerClient, error) {
	var credential azcore.TokenCredential
	var err error

	if config.AzureADClientSecret != "" {
		credential, err = azidentity.NewClientSecretCredential(config.AzureADTenantID, config.AzureADClientID, config.AzureADClientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to set up client secret Azure credentials: %s", err)
		}
	} else if config.AzureADCertificatePath != "" {
		data, err := os.ReadFile(config.AzureADCertificatePath)
		if err != nil {
			return nil, fmt.Errorf("could not read Azure AD certificate at path %s: %s", config.AzureADCertificatePath, err)
		}
		certs, key, err := azidentity.ParseCertificates(data, []byte(config.AzureADCertificatePassword))
		if err != nil {
			return nil, fmt.Errorf("could not parse Azure AD certificate: %s", err)
		}
		credential, err = azidentity.NewClientCertificateCredential(config.AzureADTenantID, config.AzureADClientID, certs, key, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to set up client secret Azure credentials: %s", err)
		}
	} else {
		var creds []azcore.TokenCredential
		var errorMessages []string
		workloadIdentityCredential, err := azidentity.NewWorkloadIdentityCredential(nil)
		if err == nil {
			creds = append(creds, workloadIdentityCredential)
		} else {
			errorMessages = append(errorMessages, "WorkloadIdentityCredential: "+err.Error())
		}
		var managedIdentityOptions *azidentity.ManagedIdentityCredentialOptions
		if config.AzureADClientID != "" {
			managedIdentityOptions = &azidentity.ManagedIdentityCredentialOptions{
				ID: azidentity.ClientID(config.AzureADClientID),
			}
		}
		managedIdentityCredential, err := azidentity.NewManagedIdentityCredential(managedIdentityOptions)
		if err == nil {
			creds = append(creds, managedIdentityCredential)
		} else {
			errorMessages = append(errorMessages, "ManagedIdentityCredential: "+err.Error())
		}
		if len(creds) == 0 {
			return nil, fmt.Errorf("failed to set up Azure credentials:\n\t" + strings.Join(errorMessages, "\n\t"))
		} else {
			credential, err = azidentity.NewChainedTokenCredential(creds, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to use default Azure credentials: %s", err)
			}
		}
	}

	consumerClient, err := azeventhubs.NewConsumerClient(config.AzureEventhubNamespace+".servicebus.windows.net", config.AzureEventhubName, azeventhubs.DefaultConsumerGroup, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to configure Event Hub: %s", err)
	}

	return consumerClient, nil
}

func getEventHubPartitionIDs(ctx context.Context, config config.ServerConfig) ([]string, error) {
	consumerClient, err := getEventHubConsumerClient(config)
	if err != nil {
		return nil, err
	}
	defer consumerClient.Close(ctx)

	info, err := consumerClient.GetEventHubProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Event Hub management node: %s", err)
	}
	return info.PartitionIDs, nil
}

func runEventHubHandlers(ctx context.Context, partitionIDs []string, logger *util.Logger, config config.ServerConfig, handler func(context.Context, *azeventhubs.ReceivedEventData)) {
	// This function keeps running until all partition clients have exited, so we can clean up the consumer client
	var wg sync.WaitGroup

	consumerClient, err := getEventHubConsumerClient(config)
	if err != nil {
		logger.PrintError("Failed to set up Azure Event Hub consumer client: %s", err)
		return
	}
	defer consumerClient.Close(ctx)

	for _, partitionID := range partitionIDs {
		wg.Add(1)
		go func(partID string) {
			defer wg.Done()

			partitionClient, err := consumerClient.NewPartitionClient(partID, &azeventhubs.PartitionClientOptions{
				StartPosition: azeventhubs.StartPosition{
					Earliest: to.Ptr(true),
				},
			})
			if err != nil {
				logger.PrintError("Failed to set up Azure Event Hub partition client for partition %s: %s", partID, err)
			}
			defer partitionClient.Close(ctx)

			for {
				events, err := partitionClient.ReceiveEvents(ctx, 1, nil)
				if err != nil {
					if err != context.Canceled {
						logger.PrintError("Failed to receive events from Azure Event Hub for partition %s: %s", partID, err)
					}
					break
				}

				for _, event := range events {
					handler(ctx, event)
				}
			}
		}(partitionID)
	}

	wg.Wait()
}

func setupEventHubReceiver(ctx context.Context, wg *sync.WaitGroup, logger *util.Logger, config config.ServerConfig, azureLogStream chan AzurePostgresLogRecord) error {
	partitionIDs, err := getEventHubPartitionIDs(ctx, config)
	if err != nil {
		return err
	}

	logger.PrintVerbose("Initializing Azure Event Hub handler for %d partitions", len(partitionIDs))

	handler := func(ctx context.Context, event *azeventhubs.ReceivedEventData) {
		var eventData AzureEventHubData
		err = json.Unmarshal(event.Body, &eventData)
		if err != nil {
			logger.PrintWarning("Error parsing Azure Event Hub event: %s", err)
		}
		for _, record := range eventData.Records {
			if record.Category == "PostgreSQLLogs" && record.OperationName == "LogEvent" {
				azureLogStream <- record
			}
		}
	}

	go runEventHubHandlers(ctx, partitionIDs, logger, config, handler)

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

func GetServerNameFromRecord(in AzurePostgresLogRecord) string {
	if in.IsSingleServer() {
		return in.LogicalServerName
	} else { // Flexible Server, Cosmos DB
		// For Flexible Server, logical server name is typically not set, so instead determine it based on the resource ID
		resourceParts := strings.Split(in.ResourceID, "/")
		return strings.ToLower(resourceParts[len(resourceParts)-1])
	}
}

// Parses one Azure Event Hub log record into one or two log lines (main + DETAIL)
func ParseRecordToLogLines(in AzurePostgresLogRecord, parser state.LogParser) ([]state.LogLine, error) {
	logLineContent := in.Properties.Message

	if in.IsSingleServer() { // Single Server
		// Adjust Azure-modified log messages to be standard Postgres log messages
		if strings.HasPrefix(logLineContent, "connection received:") {
			logLineContent = connectionReceivedRegexp.ReplaceAllString(logLineContent, "$1")
		}
		if strings.HasPrefix(logLineContent, "connection authorized:") {
			logLineContent = connectionAuthorizedRegexp.ReplaceAllString(logLineContent, "$1 $2")
		}
		if strings.HasPrefix(logLineContent, "checkpoint complete") {
			logLineContent = checkpointCompleteRegexp.ReplaceAllString(logLineContent, "$1$2")
		}
		// Add prefix and error level, which are separated from the content on
		// Single Server (but our parser expects them together)
		logLineContent = fmt.Sprintf("%s%s:  %s", in.Properties.Prefix, in.Properties.ErrorLevel, logLineContent)
	} else if in.IsCosmosDB() { // Cosmos DB
		prefix, content, ok := parser.GetPrefixAndContent(logLineContent)
		if ok {
			// Cosmos DB doesn't output the log level after the log_line_prefix and before the content
			// Manually adding it between them so the parser can parse the log line
			logLineContent = fmt.Sprintf("%s%s:  %s", prefix, in.Properties.ErrorLevel, content)
		}
	}
	logLine, ok := parser.ParseLine(logLineContent)
	if !ok {
		return []state.LogLine{}, fmt.Errorf("Can't parse log line: \"%s\"", logLineContent)
	}

	logLines := []state.LogLine{logLine}

	// DETAIL messages are not emitted in the main log stream, but instead added to the
	// primary message in the "detail" field. Create a log line to pass them along.
	//
	// Other secondary log lines (CONTEXT, STATEMENT, etc) are missing on Azure.
	if in.Properties.Detail != "" {
		detailLogLine := logLine
		detailLogLine.Content = in.Properties.Detail
		detailLogLine.LogLevel = pganalyze_collector.LogLineInformation_DETAIL
		logLines = append(logLines, detailLogLine)
	}

	return logLines, nil
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

				azureDbServerName := GetServerNameFromRecord(in)
				var server *state.Server
				for _, s := range servers {
					if azureDbServerName == s.Config.AzureDbServerName {
						server = s
					}
				}
				if server == nil {
					if globalCollectionOpts.TestRun {
						logger.PrintVerbose("Discarding log line because of unknown server (did you set the correct azure_db_server_name?): %s", azureDbServerName)
					}
					continue
				}
				parser := server.GetLogParser()
				logLines, err := ParseRecordToLogLines(in, parser)
				if err != nil {
					logger.PrintError("%s", err)
					continue
				}
				if len(logLines) == 0 {
					continue
				}

				// Ignore loglines which are outside our time window (except in test runs)
				if !logLines[0].OccurredAt.IsZero() && logLines[0].OccurredAt.Before(linesNewerThan) && !globalCollectionOpts.TestRun {
					continue
				}

				for _, logLine := range logLines {
					out <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
				}
			}
		}
	}()
}
