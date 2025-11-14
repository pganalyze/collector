package google_cloudsql

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	pubsub "cloud.google.com/go/pubsub/v2"
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
	OccurredAt       time.Time
	Content          string
	Server           *state.Server
	IsAlloyDBCluster bool
}

func setupPubSubSubscriber(ctx context.Context, wg *sync.WaitGroup, servers []*state.Server, logger *util.Logger, config config.ServerConfig, gcpLogStream chan LogStreamItem, opts state.CollectionOpts) error {
	if strings.Count(config.GcpPubsubSubscription, "/") != 3 {
		return fmt.Errorf("unsupported subscription format - must be \"projects/PROJECT_NAME/subscriptions/SUBSCRIPTION_NAME\", got: %s", config.GcpPubsubSubscription)
	}
	idParts := strings.SplitN(config.GcpPubsubSubscription, "/", 4)
	projectID := idParts[1]
	subID := idParts[3]

	var clientOpts []option.ClientOption
	if config.GcpCredentialsFile != "" {
		logger.PrintVerbose("Using GCP credentials file located at: %s", config.GcpCredentialsFile)
		clientOpts = append(clientOpts, option.WithCredentialsFile(config.GcpCredentialsFile))
	} else {
		logger.PrintVerbose("No GCP credentials file provided; assuming GKE workload identity or VM-associated service account")
	}
	client, err := pubsub.NewClient(ctx, projectID, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create Google PubSub client: %v", err)
	}

	sub := client.Subscriber(subID)
	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup, logger *util.Logger, sub *pubsub.Subscriber, maxAge time.Duration) {
		defer wg.Done()

		for {
			logger.PrintVerbose("Initializing Google Pub/Sub handler")
			err := sub.Receive(ctx, func(ctx context.Context, pubsubMsg *pubsub.Message) {
				var msg googleLogMessage
				err = json.Unmarshal(pubsubMsg.Data, &msg)
				if err != nil {
					logger.PrintError("Error parsing JSON: %s", err)
					pubsubMsg.Ack()
					return
				}

				if opts.VeryVerbose {
					jsonData, err := json.MarshalIndent(msg, "", "  ")
					if err != nil {
						logger.PrintVerbose("Failed to convert googleLogMessage struct to JSON: %v", err)
					}
					logger.PrintVerbose("Received Google Pub/Sub log data in the following format:\n")
					logger.PrintVerbose(string(jsonData))
				}

				switch msg.Resource.ResourceType {
				case "cloudsql_database":
					if !strings.HasSuffix(msg.LogName, "postgres.log") {
						pubsubMsg.Ack()
						return
					}
					databaseID, ok := msg.Resource.Labels["database_id"]
					if !ok || strings.Count(databaseID, ":") != 1 {
						pubsubMsg.Ack()
						return
					}

					parts := strings.SplitN(databaseID, ":", 2) // project_id:instance_id
					t, _ := time.Parse(time.RFC3339Nano, msg.Timestamp)

					var server *state.Server
					for _, s := range servers {
						if parts[0] == s.Config.GcpProjectID && parts[1] != "" && parts[1] == s.Config.GcpCloudSQLInstanceID {
							server = s
						}
					}

					if server == nil {
						if t.Add(maxAge).After(time.Now()) {
							// Return recent messages to be processed by a different collector
							pubsubMsg.Nack()
						} else {
							// Ack message but discard it (this causes it to be lost and cleaned up)
							pubsubMsg.Ack()
						}
						return
					}

					gcpLogStream <- LogStreamItem{
						Content:    msg.TextPayload,
						OccurredAt: t,
						Server:     server,
					}
					pubsubMsg.Ack()
					return
				case "alloydb.googleapis.com/Instance":
					if !strings.HasSuffix(msg.LogName, "postgres.log") {
						pubsubMsg.Ack()
						return
					}
					clusterID, ok := msg.Resource.Labels["cluster_id"]
					if !ok {
						pubsubMsg.Ack()
						return
					}
					instanceID, ok := msg.Resource.Labels["instance_id"]
					if !ok {
						pubsubMsg.Ack()
						return
					}
					projectID, ok := msg.Labels["CONSUMER_PROJECT"]
					if !ok {
						pubsubMsg.Ack()
						return
					}

					t, _ := time.Parse(time.RFC3339Nano, msg.Timestamp)

					var server *state.Server
					for _, s := range servers {
						if projectID == s.Config.GcpProjectID && clusterID != "" && clusterID == s.Config.GcpAlloyDBClusterID && instanceID != "" && instanceID == s.Config.GcpAlloyDBInstanceID {
							server = s
						}
					}

					if server == nil {
						if t.Add(maxAge).After(time.Now()) {
							// Return recent messages to be processed by a different collector
							pubsubMsg.Nack()
						} else {
							// Ack message but discard it (this causes it to be lost and cleaned up)
							pubsubMsg.Ack()
						}
						return
					}

					gcpLogStream <- LogStreamItem{
						Content:          msg.TextPayload,
						OccurredAt:       t,
						Server:           server,
						IsAlloyDBCluster: true,
					}
					pubsubMsg.Ack()
					return
				default:
					pubsubMsg.Ack()
					return
				}
			})
			if err == nil || err == context.Canceled {
				break
			}

			logger.PrintError("Failed to receive from Google PubSub, retrying in 1 minute: %v", err)
			time.Sleep(1 * time.Minute)
		}
	}(ctx, wg, logger, sub, config.GcpPubsubMaxAgeParsed)

	return nil
}

func SetupLogSubscriber(ctx context.Context, wg *sync.WaitGroup, opts state.CollectionOpts, logger *util.Logger, servers []*state.Server, parsedLogStream chan state.ParsedLogStreamItem) error {
	gcpLogStream := make(chan LogStreamItem, state.LogStreamBufferLen)
	setupLogTransformer(ctx, wg, servers, gcpLogStream, parsedLogStream, logger)

	// This map is used to avoid duplicate receivers to the same subscriber
	serversByPubSub := make(map[string][]*state.Server)
	for _, server := range servers {
		if server.Config.GcpPubsubSubscription != "" {
			serversByPubSub[server.Config.GcpPubsubSubscription] = append(serversByPubSub[server.Config.GcpPubsubSubscription], server)
		}
	}

	for _, s := range serversByPubSub {
		err := setupPubSubSubscriber(ctx, wg, s, logger, s[0].Config, gcpLogStream, opts)
		if err != nil {
			if opts.TestRun {
				return err
			}

			logger.PrintWarning("Skipping logs, could not setup log subscriber: %s", err)
			continue
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

				parser := in.Server.GetLogParser()
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

				if in.IsAlloyDBCluster {
					parts := alloyPrefix.FindStringSubmatch(string(logLine.Content))
					if len(parts) == 2 {
						logLine.Content = parts[1]
					}
				}
				out <- state.ParsedLogStreamItem{Identifier: in.Server.Config.Identifier, LogLine: logLine}
			}
		}
	}()
}
