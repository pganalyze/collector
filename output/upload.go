package output

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/proto"
)

func SetupSnapshotUploadForAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) {
	if opts.ForceEmptyGrant {
		return
	}
	for idx := range servers {
		go func(server *state.Server) {
			var compactLogTime time.Time
			compactLogStats := make(map[string]uint8)
			logger = logger.WithPrefixAndRememberErrors(server.Config.SectionName)
			for {
				select {
				case <-ctx.Done():
					return
				case s := <-server.FullSnapshotUpload:
					data, err := proto.Marshal(s)
					if err != nil {
						logger.PrintError("Error marshaling protocol buffers")
						continue
					}

					err = uploadViaWebsocketOrHttp(ctx, server, logger, opts.TestRun, data, s.SnapshotUuid, s.CollectedAt.AsTime(), false)
					if err != nil {
						logger.PrintError("Error uploading snapshot: %s", err)
					} else if !opts.TestRun {
						logger.PrintInfo("Submitted full snapshot successfully")
					}
				case s := <-server.CompactSnapshotUpload:
					data, err := proto.Marshal(s)
					if err != nil {
						logger.PrintError("Error marshaling protocol buffers")
						continue
					}

					err = uploadViaWebsocketOrHttp(ctx, server, logger, opts.TestRun, data, s.SnapshotUuid, s.CollectedAt.AsTime(), false)
					if err != nil {
						logger.PrintError("Error uploading snapshot: %s", err)
						continue
					}
					if opts.TestRun {
						continue
					}

					kind := kindFromCompactSnapshot(s)
					logger.PrintVerbose("Submitted compact %s snapshot successfully", kind)
					compactLogStats[kind] = compactLogStats[kind] + 1
					if compactLogTime.IsZero() {
						compactLogTime = time.Now().Truncate(time.Minute)
					} else if time.Since(compactLogTime) > time.Minute {
						details := summarizeCounts(compactLogStats)
						if len(details) > 0 {
							logger.PrintInfo("Submitted compact snapshots successfully: " + details)
						}
						compactLogTime = time.Now().Truncate(time.Minute)
						compactLogStats = make(map[string]uint8)
					}
				}
			}
		}(servers[idx])
	}
}

func summarizeCounts(counts map[string]uint8) string {
	var keys []string
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	details := ""
	for i, kind := range keys {
		details += fmt.Sprintf("%d %s", counts[kind], kind)
		if i < len(keys)-1 {
			details += ", "
		}
	}
	return details
}

func uploadViaWebsocketOrHttp(ctx context.Context, server *state.Server, logger *util.Logger, testRun bool, data []byte, snapshotUUID string, collectedAt time.Time, compactSnapshot bool) error {
	var compressedData bytes.Buffer
	w := zlib.NewWriter(&compressedData)
	w.Write(data)
	w.Close()

	if server.WebSocket.Load() != nil {
		server.SnapshotStream <- compressedData.Bytes()
	} else {
		s3Location, err := uploadSnapshot(ctx, server.Config.HTTPClientWithRetry, server.Grant.Load(), logger, compressedData.Bytes(), snapshotUUID)
		if err != nil {
			return err
		}
		submitSnapshot(ctx, server, testRun, logger, s3Location, collectedAt, compactSnapshot)
	}
	return nil
}
