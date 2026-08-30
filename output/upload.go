package output

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
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
	for _, server := range servers {
		go snapshotUploadForServer(ctx, server, logger.WithPrefixAndRememberErrors(server.Config.SectionName), opts)
	}
}

// Maximum time a legacy HTTP snapshot upload may take before it is abandoned,
// so a slow or unreachable server does not indefinitely stall the per-server
// upload queue (and with it all snapshot types, since uploads run sequentially)
const snapshotUploadTimeout = 1 * time.Minute

func snapshotUploadForServer(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts) {
	var compactLogTime time.Time
	compactLogStats := make(map[string]uint8)
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

			err = uploadViaWebsocketOrHttp(ctx, server, logger, opts, data, s.SnapshotUuid, snapshotUploadTimeout)
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

			err = uploadViaWebsocketOrHttp(ctx, server, logger, opts, data, s.SnapshotUuid, snapshotUploadTimeout)
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

func uploadViaWebsocketOrHttp(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts, data []byte, snapshotUUID string, httpUploadTimeout time.Duration) error {
	var compressedData bytes.Buffer
	w := zlib.NewWriter(&compressedData)
	w.Write(data)
	w.Close()

	if server.WebSocket.Connected() {
		logger.PrintVerbose("Uploading snapshot to websocket")
		server.WebSocket.Write <- compressedData.Bytes()
	} else if server.Config.APIRequireWebsocket {
		return errors.New("Error uploading snapshot: WebSocket not connected")
	} else {
		uploadCtx, cancel := context.WithTimeout(ctx, httpUploadTimeout)
		defer cancel()
		_, err := uploadSnapshot(uploadCtx, server.Config.HTTPClientWithRetry, server.Grant.Load(), logger, compressedData.Bytes(), snapshotUUID)
		if err != nil {
			return err
		}
		if opts.TestRun {
			return markTestRun(uploadCtx, server, opts)
		}
	}
	return nil
}
