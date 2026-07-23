package output

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func SetupSnapshotUploadForAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) {
	if opts.ForceEmptyGrant {
		return
	}
	for _, server := range servers {
		prefixedLogger := logger.WithPrefixAndRememberErrors(server.Config.SectionName)
		server.SnapshotQueue.Logger = prefixedLogger
		go snapshotUploadForServer(ctx, server, prefixedLogger, opts)
	}
}

func snapshotUploadForServer(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts) {
	var compactLogTime time.Time
	var compactLogStats = make(map[string]uint8)
	var failed bool
	var delay time.Duration

	for {
		if failed {
			delay = min(delay*5+10*time.Millisecond, 10*time.Second)
		} else {
			delay = 10 * time.Millisecond // Small delay to avoid high CPU usage in loop
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		tx, err := server.SnapshotQueue.Pop(ctx)
		if err != nil {
			continue
		}

		err = uploadViaWebsocketOrHttp(ctx, server, logger, opts, tx.Snapshot)
		if err != nil {
			logger.PrintError("Error uploading %s snapshot: %s", tx.Kind, err)
			tx.Rollback()
			failed = true
		} else {
			tx.Commit()
			failed = false
			if !opts.TestRun {
				logger.PrintInfo("Submitted %s snapshot successfully", tx.Kind)
			}
			if tx.Kind == "full" {
				continue
			}
			// Compact snapshot: log stats periodically
			kind := tx.Kind
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

func uploadViaWebsocketOrHttp(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts, data []byte) error {
	if server.WebSocket.Connected() {
		logger.PrintVerbose("Uploading snapshot to websocket")
		result := make(chan error, 1)
		select {
		case server.WebSocket.Write <- util.WriteRequest{Data: data, Result: result}:
			select {
			case err := <-result:
				if err != nil {
					return fmt.Errorf("WebSocket write failed: %w", err)
				}
				return nil
			case <-time.After(5 * time.Second):
				logger.PrintWarning("WebSocket write timed out, falling back to HTTP")
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-time.After(5 * time.Second):
			logger.PrintWarning("WebSocket write timed out, falling back to HTTP")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if server.Config.APIRequireWebsocket {
		return errors.New("Error uploading snapshot: WebSocket not connected")
	}
	return uploadSnapshot(ctx, server.Config.HTTPClient, server.Grant.Load(), logger, data)
}
