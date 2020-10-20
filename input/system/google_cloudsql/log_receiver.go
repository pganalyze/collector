package google_cloudsql

import (
	"context"
	"time"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/logs"
	"github.com/pganalyze/collector/logs/stream"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

type LogStreamItem struct {
	GcpProjectID          string
	GcpCloudSQLInstanceID string
	OccurredAt            time.Time
	Content               string
}

func SetupLogReceiver(ctx context.Context, servers []*state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger, gcpLogStream <-chan LogStreamItem) {
	logReceiver(ctx, servers, gcpLogStream, globalCollectionOpts, logger, nil)
}

func logReceiver(ctx context.Context, servers []*state.Server, in <-chan LogStreamItem, globalCollectionOpts state.CollectionOpts, logger *util.Logger, logTestSucceeded chan<- bool) {
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

				// We ignore failures here since we want the per-backend stitching logic
				// that runs later on (and any other parsing errors will just be ignored)
				logLine, _ := logs.ParseLogLineWithPrefix("", in.Content)
				logLine.CollectedAt = time.Now()
				logLine.OccurredAt = in.OccurredAt
				logLine.UUID = uuid.NewV4()

				// Ignore loglines which are outside our time window
				nullTime := time.Time{}
				if logLine.OccurredAt != nullTime && logLine.OccurredAt.Before(linesNewerThan) {
					continue
				}

				for _, server := range servers {
					if in.GcpProjectID == server.Config.GcpProjectID && in.GcpCloudSQLInstanceID == server.Config.GcpCloudSQLInstanceID {
						identifier := server.Config.Identifier
						prefixedLogger := logger.WithPrefix(server.Config.SectionName)
						logLinesByServer[identifier] = append(logLinesByServer[identifier], logLine)
						logLinesByServer[identifier] = stream.ProcessLogStream(server, logLinesByServer[identifier], globalCollectionOpts, prefixedLogger, logTestSucceeded, stream.LogTestCollectorIdentify)
					}
				}

			case <-timeout:
				for identifier := range logLinesByServer {
					if len(logLinesByServer[identifier]) > 0 {
						server := &state.Server{}
						for _, s := range servers {
							if s.Config.Identifier == identifier {
								server = s
							}
						}
						prefixedLogger := logger.WithPrefix(server.Config.SectionName)
						logLinesByServer[identifier] = stream.ProcessLogStream(server, logLinesByServer[identifier], globalCollectionOpts, prefixedLogger, logTestSucceeded, stream.LogTestCollectorIdentify)
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
