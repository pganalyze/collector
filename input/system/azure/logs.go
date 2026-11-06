package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
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

func getAzureCredential(config config.ServerConfig) (azcore.TokenCredential, error) {
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
			return nil, fmt.Errorf("failed to set up Azure credentials:\n\t%s", strings.Join(errorMessages, "\n\t"))
		} else {
			credential, err = azidentity.NewChainedTokenCredential(creds, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to use default Azure credentials: %s", err)
			}
		}
	}
	return credential, nil
}

// Wraps a credential so that token requests also get cancelled when the
// given context is cancelled.
//
// The Event Hub client calls GetToken with context.TODO() (see
// azeventhubs/internal/sbauth/token_provider.go), so token requests are not
// cancelled when the context passed to Event Hub client methods is. Since the
// client holds a lock while requesting a token for a new link, this would also
// block closing the client, potentially for minutes (e.g. with IMDS retries).
type contextTokenCredential struct {
	ctx        context.Context
	credential azcore.TokenCredential
}

func (c contextTokenCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(c.ctx, cancel)
	defer stop()
	return c.credential.GetToken(ctx, opts)
}

func getEventHubConsumerClient(ctx context.Context, config config.ServerConfig) (*azeventhubs.ConsumerClient, error) {
	credential, err := getAzureCredential(config)
	if err != nil {
		return nil, err
	}
	credential = contextTokenCredential{ctx: ctx, credential: credential}

	consumerClient, err := azeventhubs.NewConsumerClient(config.AzureEventhubNamespace+".servicebus.windows.net", config.AzureEventhubName, azeventhubs.DefaultConsumerGroup, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to configure Event Hub: %s", err)
	}

	return consumerClient, nil
}

func getEventHubPartitionIDs(ctx context.Context, consumerClient *azeventhubs.ConsumerClient) ([]string, error) {
	info, err := consumerClient.GetEventHubProperties(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Event Hub management node: %s", err)
	}
	return info.PartitionIDs, nil
}

// Tracks the sequence number of the last event received for each partition,
// so we can resume where we left off after reconnecting (instead of reading
// all events retained in the Event Hub again)
type eventHubPartitionPositions struct {
	mu        sync.Mutex
	positions map[string]int64
}

func (p *eventHubPartitionPositions) startPosition(partitionID string) azeventhubs.StartPosition {
	p.mu.Lock()
	defer p.mu.Unlock()
	if seq, ok := p.positions[partitionID]; ok {
		return azeventhubs.StartPosition{SequenceNumber: to.Ptr(seq), Inclusive: false}
	}
	// Older log lines get filtered out by the log transformer
	return azeventhubs.StartPosition{Earliest: to.Ptr(true)}
}

func (p *eventHubPartitionPositions) update(partitionID string, seq int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.positions[partitionID] = seq
}

// Receives events from all partitions of the Event Hub until the context is
// cancelled, or an error occurs on any partition. Partial failures cause all
// partitions to stop, so that the caller can reconnect from scratch.
//
// Also returns whether any events were received, so the caller can tell a
// working connection that failed later apart from one that never worked.
// Note that connection problems (e.g. missing permissions or quota limits)
// may only surface when receiving, since partition links are opened lazily.
func receiveEventHubEvents(ctx context.Context, logger *util.Logger, config config.ServerConfig, positions *eventHubPartitionPositions, handler func(context.Context, *azeventhubs.ReceivedEventData)) (bool, error) {
	// Cancelled when any partition fails, so that all partitions stop. This is
	// also used for the consumer client, so that token requests for other
	// partitions are cancelled as well, instead of delaying the reconnect
	receiveCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	consumerClient, err := getEventHubConsumerClient(receiveCtx, config)
	if err != nil {
		return false, err
	}
	defer consumerClient.Close(ctx)

	partitionIDs, err := getEventHubPartitionIDs(receiveCtx, consumerClient)
	if err != nil {
		return false, err
	}

	logger.PrintVerbose("Initializing Azure Event Hub handler for %d partitions", len(partitionIDs))

	var receivedEvents atomic.Bool
	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	fail := func(err error) {
		errOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	for _, partitionID := range partitionIDs {
		wg.Add(1)
		go func(partID string) {
			defer wg.Done()

			partitionClient, err := consumerClient.NewPartitionClient(partID, &azeventhubs.PartitionClientOptions{
				StartPosition: positions.startPosition(partID),
			})
			if err != nil {
				fail(fmt.Errorf("failed to set up partition client for partition %s: %s", partID, err))
				return
			}
			defer partitionClient.Close(receiveCtx)

			for {
				// Stop right away once cancelled. Receiving with a cancelled
				// context still returns events that were already prefetched,
				// one call at a time, so we'd otherwise work through the whole
				// prefetch buffer before exiting. Events left behind are
				// received again after reconnecting, since the position is
				// only advanced for events that were handled.
				if receiveCtx.Err() != nil {
					return
				}

				events, err := partitionClient.ReceiveEvents(receiveCtx, 1, nil)
				if err != nil {
					if receiveCtx.Err() == nil {
						fail(fmt.Errorf("failed to receive events for partition %s: %s", partID, err))
					}
					return
				}
				for _, event := range events {
					// Pass the parent context, not the receive context, so that
					// handling an event we already received isn't cut short
					// when another partition fails
					handler(ctx, event)
					positions.update(partID, event.SequenceNumber)
					receivedEvents.Store(true)
				}
			}
		}(partitionID)
	}

	wg.Wait()

	return receivedEvents.Load(), firstErr
}

const (
	eventHubMinRetryDelay = 10 * time.Second
	eventHubMaxRetryDelay = 5 * time.Minute
)

// Keeps receiving events from the Event Hub until the context is cancelled,
// reconnecting with exponential backoff whenever setup or receiving fails.
func runEventHubReceiver(ctx context.Context, logger *util.Logger, config config.ServerConfig, handler func(context.Context, *azeventhubs.ReceivedEventData)) {
	positions := &eventHubPartitionPositions{positions: make(map[string]int64)}
	retryDelay := eventHubMinRetryDelay
	var err error

	for {
		if err != nil {
			logger.PrintError("Azure Event Hub log receiver failed, retrying in %s: %s", retryDelay, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
			retryDelay = min(retryDelay*2, eventHubMaxRetryDelay)
		}

		var receivedEvents bool
		receivedEvents, err = receiveEventHubEvents(ctx, logger, config, positions, handler)
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			// Should not happen unless the context was cancelled, but make sure we don't spin
			err = fmt.Errorf("receiver stopped unexpectedly")
		}
		// Only start over with the minimum delay if the connection actually
		// worked, so that persistent errors keep backing off
		if receivedEvents {
			retryDelay = eventHubMinRetryDelay
		}
	}
}

func setupEventHubReceiver(ctx context.Context, wg *sync.WaitGroup, logger *util.Logger, config config.ServerConfig, azureLogStream chan AzurePostgresLogRecord, opts state.CollectionOpts) error {
	// For test runs, verify that we can connect, so that errors fail the test
	// run. Otherwise the receiver connects in the background and keeps
	// retrying, since errors may be temporary (e.g. a role assignment for a
	// managed identity that has not propagated yet), and we don't want to
	// hold up startup while the connection is attempted.
	if opts.TestRun {
		consumerClient, err := getEventHubConsumerClient(ctx, config)
		if err != nil {
			return err
		}
		_, err = getEventHubPartitionIDs(ctx, consumerClient)
		consumerClient.Close(ctx)
		if err != nil {
			return err
		}
	}

	handler := func(ctx context.Context, event *azeventhubs.ReceivedEventData) {
		var eventData AzureEventHubData
		err := json.Unmarshal(event.Body, &eventData)
		if err != nil {
			logger.PrintWarning("Error parsing Azure Event Hub event: %s", err)
			return
		}
		if opts.VeryVerbose {
			jsonData, err := json.MarshalIndent(eventData, "", "  ")
			if err != nil {
				logger.PrintVerbose("Failed to convert AzureEventHubData struct to JSON: %v", err)
			}
			logger.PrintVerbose("Received Azure EventHub log data in the following format:\n")
			logger.PrintVerbose(string(jsonData))
		}
		for _, record := range eventData.Records {
			if record.Category == "PostgreSQLLogs" && record.OperationName == "LogEvent" {
				// The log transformer stops reading when the context is cancelled, so
				// don't block forever if the buffer is full
				select {
				case azureLogStream <- record:
				case <-ctx.Done():
					return
				}
			}
		}
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		runEventHubReceiver(ctx, logger, config, handler)
	}()

	return nil
}

func SetupLogSubscriber(ctx context.Context, wg *sync.WaitGroup, opts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) error {
	azureLogStream := make(chan AzurePostgresLogRecord, state.LogStreamBufferLen)
	setupLogTransformer(ctx, wg, servers, azureLogStream, parsedLogStream, opts, logger)

	// This map is used to avoid duplicate receivers to the same Azure Event Hub
	eventHubReceivers := make(map[string]bool)

	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)
		if server.Config.AzureEventhubNamespace != "" && server.Config.AzureEventhubName != "" {
			if _, ok := eventHubReceivers[server.Config.AzureEventhubNamespace+"/"+server.Config.AzureEventhubName]; ok {
				continue
			}
			err := setupEventHubReceiver(ctx, wg, prefixedLogger, server.Config, azureLogStream, opts)
			if err != nil {
				if opts.TestRun {
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

	if in.IsSingleServer() {
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
	} else if in.IsCosmosDB() {
		prefix, content, ok := parser.GetPrefixAndContent(logLineContent)
		if ok {
			// Cosmos DB doesn't output the log level after the log_line_prefix and before the content
			// Manually adding it between them so the parser can parse the log line
			logLineContent = fmt.Sprintf("%s%s:  %s", prefix, in.Properties.ErrorLevel, content)
		}
	} else {
		// Flexible Server: no tweak needed
	}
	logLine, ok := parser.ParseLine(logLineContent)
	if !ok {
		return []state.LogLine{}, fmt.Errorf("can't parse log line: \"%s\"", logLineContent)
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

func setupLogTransformer(ctx context.Context, wg *sync.WaitGroup, servers []*state.Server, in <-chan AzurePostgresLogRecord, out chan state.ParsedLogStreamItem, opts state.CollectionOpts, logger *util.Logger) {
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
					if opts.TestRun {
						logger.PrintVerbose("Discarding log line because of unknown server (did you set the correct azure_db_server_name?): %s", azureDbServerName)
					}
					continue
				}
				parser := server.GetLogParser()
				if parser == nil {
					continue
				}
				logLines, err := ParseRecordToLogLines(in, parser)
				if err != nil {
					logger.PrintError("%s", err)
					continue
				}
				if len(logLines) == 0 {
					continue
				}

				// Ignore loglines which are outside our time window (except in test runs)
				if !logLines[0].OccurredAt.IsZero() && logLines[0].OccurredAt.Before(linesNewerThan) && !opts.TestRun {
					continue
				}

				for _, logLine := range logLines {
					out <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
				}
			}
		}
	}()
}
