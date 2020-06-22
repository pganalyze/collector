package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/logs/stream"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
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

func SetupLogReceiver(ctx context.Context, servers []state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger, azureLogStream <-chan AzurePostgresLogRecord) {
	logReceiver(ctx, servers, azureLogStream, globalCollectionOpts, logger, nil)
}

func logReceiver(ctx context.Context, servers []state.Server, in <-chan AzurePostgresLogRecord, globalCollectionOpts state.CollectionOpts, logger *util.Logger, logTestSucceeded chan<- bool) {
	go func() {
		logLinesByServer := make(map[config.ServerIdentifier][]state.LogLine)

		// Only ingest log lines that were written in the last minute before startup
		linesNewerThan := time.Now().Add(-1 * time.Minute)

		// Use a timeout to clear out loglines that don't have any follow-on lines
		// (the threshold used in stream.ProcessLogStream is 3 seconds)
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(3 * time.Second)
			timeout <- true
		}()

		for {
			select {
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
				nullTime := time.Time{}
				if logLine.OccurredAt != nullTime && logLine.OccurredAt.Before(linesNewerThan) && !globalCollectionOpts.TestRun {
					continue
				}

				for _, server := range servers {
					if in.LogicalServerName == server.Config.AzureDbServerName {
						identifier := server.Config.Identifier
						prefixedLogger := logger.WithPrefix(server.Config.SectionName)
						logLinesByServer[identifier] = append(logLinesByServer[identifier], logLine)
						logLinesByServer[identifier] = stream.ProcessLogStream(server, logLinesByServer[identifier], globalCollectionOpts, prefixedLogger, logTestSucceeded, stream.LogTestAnyEvent)
					}
				}

			case <-timeout:
				for identifier := range logLinesByServer {
					if len(logLinesByServer[identifier]) > 0 {
						server := state.Server{}
						for _, s := range servers {
							if s.Config.Identifier == identifier {
								server = s
							}
						}
						prefixedLogger := logger.WithPrefix(server.Config.SectionName)
						logLinesByServer[identifier] = stream.ProcessLogStream(server, logLinesByServer[identifier], globalCollectionOpts, prefixedLogger, logTestSucceeded, stream.LogTestAnyEvent)
					}
				}
				go func() {
					time.Sleep(3 * time.Second)
					timeout <- true
				}()
			case <-ctx.Done():
				return
			}
		}
	}()
}
