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
	"github.com/pganalyze/collector/output/pganalyze_collector"
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
				azureLogStream <- record
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

// Parses one Azure Event Hub log record into one or two log lines (main + DETAIL)
func ParseRecordToLogLines(in AzurePostgresLogRecord) ([]state.LogLine, string, error) {
	var azureDbServerName string

	logLineContent := in.Properties.Message

	if in.LogicalServerName == "" { // Flexible Server
		// For Flexible Server, logical server name is not set, so instead determine it based on the resource ID
		resourceParts := strings.Split(in.ResourceID, "/")
		azureDbServerName = strings.ToLower(resourceParts[len(resourceParts)-1])
	} else { // Single Server
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

		azureDbServerName = in.LogicalServerName
	}

	logLine, ok := logs.ParseLogLineWithPrefix("", logLineContent)
	if !ok {
		return []state.LogLine{}, "", fmt.Errorf("Can't parse log line: \"%s\"", logLineContent)
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

	return logLines, azureDbServerName, nil
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

				logLines, azureDbServerName, err := ParseRecordToLogLines(in)
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

				foundServer := false
				for _, server := range servers {
					if azureDbServerName == server.Config.AzureDbServerName {
						foundServer = true

						for _, logLine := range logLines {
							logLine.CollectedAt = time.Now()
							logLine.UUID = uuid.NewV4()
							out <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
						}
					}
				}

				if !foundServer && globalCollectionOpts.TestRun {
					logger.PrintError("Discarding log line because of unknown server (did you set the correct azure_db_server_name?): %s", in.LogicalServerName)
				}
			}
		}
	}()
}
