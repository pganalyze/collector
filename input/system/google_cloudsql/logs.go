package google_cloudsql

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

type googleLogResource struct {
	ResourceType string            `json:"type"`
	Labels       map[string]string `json:"labels"`
}

type googleLogMessage struct {
	InsertID         string            `json:"insertId"`
	LogName          string            `json:"logName"`
	ReceiveTimestamp string            `json:"receiveTimestamp"`
	Resource         googleLogResource `json:"resource"`
	Severity         string            `json:"severity"`
	TextPayload      string            `json:"textPayload"`
	Timestamp        string            `json:"timestamp"`
	Labels           map[string]string `json:"labels"`
}

type LogStreamItem struct {
	GcpProjectID          string
	GcpCloudSQLInstanceID string
	GcpAlloyDBClusterID   string
	GcpAlloyDBInstanceID  string
	OccurredAt            time.Time
	Content               string
}

func setupPubSubSubscriber(ctx context.Context, wg *sync.WaitGroup, logger *util.Logger, config config.ServerConfig, gcpLogStream chan LogStreamItem) error {
	if strings.Count(config.GcpPubsubSubscription, "/") != 3 {
		return fmt.Errorf("Unsupported subscription format - must be \"projects/PROJECT_NAME/subscriptions/SUBSCRIPTION_NAME\", got: %s", config.GcpPubsubSubscription)
	}
	idParts := strings.SplitN(config.GcpPubsubSubscription, "/", 4)
	projectID := idParts[1]
	subID := idParts[3]

	var opts []option.ClientOption
	if config.GcpCredentialsFile != "" {
		logger.PrintVerbose("Using GCP credentials file located at: %s", config.GcpCredentialsFile)
		opts = append(opts, option.WithCredentialsFile(config.GcpCredentialsFile))
	} else {
		logger.PrintVerbose("No GCP credentials file provided; assuming GKE workload identity or VM-associated service account")
	}
	client, err := pubsub.NewClient(ctx, projectID, opts...)
	if err != nil {
		return fmt.Errorf("Failed to create Google PubSub client: %v", err)
	}

	sub := client.Subscription(subID)
	go func(ctx context.Context, wg *sync.WaitGroup, logger *util.Logger, sub *pubsub.Subscription) {
		wg.Add(1)

		// Debug logging for Google Pub/Sub messages
		f, err := os.Open("/log/pganalyze_google_cloudsql_debug.log")
		if err != nil {
			logger.PrintError("Failed to open file for logging: %s", err)
			return
		}
		defer f.Close()

		for {
			logger.PrintVerbose("Initializing Google Pub/Sub handler")
			err := sub.Receive(ctx, func(ctx context.Context, pubsubMsg *pubsub.Message) {
				pubsubMsg.Ack()

				var msg googleLogMessage
				err = json.Unmarshal(pubsubMsg.Data, &msg)
				if err != nil {
					logger.PrintError("Error parsing JSON: %s", err)
					return
				}

				log_msg := fmt.Sprintf("%+v", msg)
				f.WriteString(log_msg)
				// End of debug logging

				if msg.Resource.ResourceType == "cloudsql_database" {
					if !strings.HasSuffix(msg.LogName, "postgres.log") {
						return
					}
					databaseID, ok := msg.Resource.Labels["database_id"]
					if !ok || strings.Count(databaseID, ":") != 1 {
						return
					}

					parts := strings.SplitN(databaseID, ":", 2) // project_id:instance_id
					t, _ := time.Parse(time.RFC3339Nano, msg.Timestamp)

					gcpLogStream <- LogStreamItem{
						GcpProjectID:          parts[0],
						GcpCloudSQLInstanceID: parts[1],
						Content:               msg.TextPayload,
						OccurredAt:            t,
					}
					return
				} else if msg.Resource.ResourceType == "alloydb.googleapis.com/Instance" {
					if !strings.HasSuffix(msg.LogName, "postgres.log") {
						return
					}
					clusterID, ok := msg.Resource.Labels["cluster_id"]
					if !ok {
						return
					}
					instanceID, ok := msg.Resource.Labels["instance_id"]
					if !ok {
						return
					}
					projectID, ok := msg.Labels["CONSUMER_PROJECT"]
					if !ok {
						return
					}

					t, _ := time.Parse(time.RFC3339Nano, msg.Timestamp)

					gcpLogStream <- LogStreamItem{
						GcpProjectID:         projectID,
						GcpAlloyDBClusterID:  clusterID,
						GcpAlloyDBInstanceID: instanceID,
						Content:              msg.TextPayload,
						OccurredAt:           t,
					}
					return
				}
			})
			if err == nil || err == context.Canceled {
				break
			}

			logger.PrintError("Failed to receive from Google PubSub, retrying in 1 minute: %v", err)
			time.Sleep(1 * time.Minute)
		}
		wg.Done()
	}(ctx, wg, logger, sub)

	return nil
}

func SetupLogSubscriber(ctx context.Context, wg *sync.WaitGroup, opts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) error {
	gcpLogStream := make(chan LogStreamItem, state.LogStreamBufferLen)
	setupLogTransformer(ctx, wg, servers, gcpLogStream, parsedLogStream, logger)

	// This map is used to avoid duplicate receivers to the same subscriber
	gcpPubSubHandlers := make(map[string]bool)

	for _, server := range servers {
		prefixedLogger := logger.WithPrefix(server.Config.SectionName)
		if server.Config.GcpPubsubSubscription != "" {
			_, ok := gcpPubSubHandlers[server.Config.GcpPubsubSubscription]
			if ok {
				continue
			}
			err := setupPubSubSubscriber(ctx, wg, prefixedLogger, server.Config, gcpLogStream)
			if err != nil {
				if opts.TestRun {
					return err
				}

				prefixedLogger.PrintWarning("Skipping logs, could not setup log subscriber: %s", err)
				continue
			}

			gcpPubSubHandlers[server.Config.GcpPubsubSubscription] = true
		}
	}

	return nil
}

// AlloyDB adds a special [filename:lineno] prefix to all log lines (not part of log_line_prefix)
var alloyPrefix = regexp.MustCompile(`(?s)^\[[\w.-]+:\d+\]  (.*)`)

func setupLogTransformer(ctx context.Context, wg *sync.WaitGroup, servers []*state.Server, in <-chan LogStreamItem, out chan state.ParsedLogStreamItem, logger *util.Logger) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Only ingest log lines that were written in the last minute before startup
		linesNewerThan := time.Now().Add(-1 * time.Minute)

		for {
			select {
			case <-ctx.Done():
				return
			case in, ok := <-in:
				if !ok {
					return
				}

				var server *state.Server
				var isAlloyDBCluster bool

				for _, s := range servers {
					if in.GcpProjectID == s.Config.GcpProjectID && in.GcpCloudSQLInstanceID != "" && in.GcpCloudSQLInstanceID == s.Config.GcpCloudSQLInstanceID {
						server = s
					}
					if in.GcpProjectID == s.Config.GcpProjectID && in.GcpAlloyDBClusterID != "" && in.GcpAlloyDBClusterID == s.Config.GcpAlloyDBClusterID && in.GcpAlloyDBInstanceID != "" && in.GcpAlloyDBInstanceID == s.Config.GcpAlloyDBInstanceID {
						server = s
						isAlloyDBCluster = true
					}
				}

				if server == nil {
					continue
				}

				parser := server.GetLogParser()
				if parser == nil {
					continue
				}
				// We ignore failures here since we want the per-backend stitching logic
				// that runs later on (and any other parsing errors will just be ignored).
				// Note that we need to restore the original trailing newlines since
				// AnalyzeStreamInGroups expects them and they are not present in the GCP
				// log stream.
				logLine, _ := parser.ParseLine(in.Content + "\n")
				logLine.OccurredAt = in.OccurredAt

				// Ignore loglines which are outside our time window
				if !logLine.OccurredAt.IsZero() && logLine.OccurredAt.Before(linesNewerThan) {
					continue
				}

				if isAlloyDBCluster {
					parts := alloyPrefix.FindStringSubmatch(string(logLine.Content))
					if len(parts) == 2 {
						logLine.Content = parts[1]
					}
				}
				out <- state.ParsedLogStreamItem{Identifier: server.Config.Identifier, LogLine: logLine}
			}
		}
	}()
}
