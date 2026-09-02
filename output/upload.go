package output

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
)

func SetupSnapshotUploadForAllServers(ctx context.Context, servers []*state.Server, opts state.CollectionOpts, logger *util.Logger) {
	if opts.ForceEmptyGrant {
		return
	}
	for _, server := range servers {
		go snapshotUploadForServer(ctx, server, logger.WithPrefixAndRememberErrors(server.Config.SectionName), opts)
	}
}

func snapshotUploadForServer(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts) {
	var compactLogTime time.Time
	compactLogStats := make(map[string]uint8)
	for {
		select {
		case <-ctx.Done():
			return
		case s := <-server.FullSnapshotUpload:
			data, err := marshalSnapshot(s)
			if err != nil {
				logger.PrintError("Error marshaling protocol buffers: %s", err)
				continue
			}

			err = uploadViaWebsocketOrHttp(ctx, server, logger, opts, data, s.SnapshotUuid, s.CollectedAt.AsTime(), false)
			if err != nil {
				logger.PrintError("Error uploading snapshot: %s", err)
			} else if !opts.TestRun {
				logger.PrintInfo("Submitted full snapshot successfully")
			}
		case s := <-server.CompactSnapshotUpload:
			data, err := marshalSnapshot(s)
			if err != nil {
				logger.PrintError("Error marshaling protocol buffers: %s", err)
				continue
			}

			err = uploadViaWebsocketOrHttp(ctx, server, logger, opts, data, s.SnapshotUuid, s.CollectedAt.AsTime(), false)
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

// Marshal a snapshot, annotating any error with the paths of invalid UTF-8 fields.
func marshalSnapshot(m proto.Message) ([]byte, error) {
	data, err := proto.Marshal(m)
	return data, annotateInvalidUTF8(err, m)
}

// proto.Marshal and protojson.Marshal reject strings that aren't valid UTF-8, but the
// error they return doesn't say which field was affected. To make that tractable to
// track down (and fix at the source, e.g. by adding a setting to the denylist in
// output/transform), walk the message on failure and append the offending paths.
func annotateInvalidUTF8(err error, m proto.Message) error {
	if err == nil {
		return nil
	}
	if paths := findInvalidUTF8(m); len(paths) > 0 {
		return fmt.Errorf("%w (in %s)", err, strings.Join(paths, ", "))
	}
	return err
}

// Returns the path of every string field, list element, map value and map key in the
// message that isn't valid UTF-8, e.g. ".settings[0].boot_value.value".
func findInvalidUTF8(m proto.Message) []string {
	var paths []string
	protorange.Range(m.ProtoReflect(), func(p protopath.Values) error {
		last := p.Index(-1)
		// p.Path[0] is the root step, which would print as the message type name
		path := p.Path[1:].String()
		if s, ok := last.Value.Interface().(string); ok && !utf8.ValidString(s) {
			paths = append(paths, path)
		}
		if last.Step.Kind() == protopath.MapIndexStep {
			if k, ok := last.Step.MapIndex().Interface().(string); ok && !utf8.ValidString(k) {
				paths = append(paths, path+" (key)")
			}
		}
		return nil
	})
	return paths
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

func uploadViaWebsocketOrHttp(ctx context.Context, server *state.Server, logger *util.Logger, opts state.CollectionOpts, data []byte, snapshotUUID string, collectedAt time.Time, compactSnapshot bool) error {
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
		s3Location, err := uploadSnapshot(ctx, server.Config.HTTPClientWithRetry, server.Grant.Load(), logger, compressedData.Bytes(), snapshotUUID)
		if err != nil {
			return err
		}
		submitSnapshot(ctx, server, opts, logger, s3Location, collectedAt, compactSnapshot)
	}
	return nil
}
