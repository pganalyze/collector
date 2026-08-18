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
	"google.golang.org/protobuf/reflect/protoreflect"
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
			data, err := marshalSnapshot(s, logger)
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
			data, err := marshalSnapshot(s, logger)
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

// The Unicode replacement character (U+FFFD) substituted for runs
// of invalid bytes, so an unrepresentable field becomes a valid (if lossy) string.
const utf8Replacement = "\uFFFD"

// Marshal a snapshot. If it fails due to invalid UTF-8 in a string field, scrub
// the offending bytes and retry once.
func marshalSnapshot(m proto.Message, logger *util.Logger) ([]byte, error) {
	data, err := proto.Marshal(m)
	if err != nil {
		sanitizeInvalidUTF8(m.ProtoReflect())
		data, err = proto.Marshal(m)
		if err == nil {
			logger.PrintVerbose("Sanitized invalid UTF-8 in a snapshot string field before marshaling")
		}
	}
	return data, err
}

// Recursively replaces invalid UTF-8 in every string field, list element, and map key/value
// of a proto message. Mutations to scalar string fields are deferred until after the range,
// since protoreflect only permits mutating the current field's own value during iteration.
func sanitizeInvalidUTF8(msg protoreflect.Message) {
	var fixFds []protoreflect.FieldDescriptor
	var fixVals []string
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList():
			sanitizeUTF8List(fd, v.List())
		case fd.IsMap():
			sanitizeUTF8Map(fd, v.Map())
		case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
			sanitizeInvalidUTF8(v.Message())
		case fd.Kind() == protoreflect.StringKind:
			if s := v.String(); !utf8.ValidString(s) {
				fixFds = append(fixFds, fd)
				fixVals = append(fixVals, strings.ToValidUTF8(s, utf8Replacement))
			}
		}
		return true
	})
	for i, fd := range fixFds {
		msg.Set(fd, protoreflect.ValueOfString(fixVals[i]))
	}
}

func sanitizeUTF8List(fd protoreflect.FieldDescriptor, list protoreflect.List) {
	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		for i := 0; i < list.Len(); i++ {
			sanitizeInvalidUTF8(list.Get(i).Message())
		}
	case protoreflect.StringKind:
		for i := 0; i < list.Len(); i++ {
			if s := list.Get(i).String(); !utf8.ValidString(s) {
				list.Set(i, protoreflect.ValueOfString(strings.ToValidUTF8(s, utf8Replacement)))
			}
		}
	}
}

func sanitizeUTF8Map(fd protoreflect.FieldDescriptor, m protoreflect.Map) {
	valFd := fd.MapValue()
	valIsMessage := valFd.Kind() == protoreflect.MessageKind || valFd.Kind() == protoreflect.GroupKind
	valIsString := valFd.Kind() == protoreflect.StringKind
	keyIsString := fd.MapKey().Kind() == protoreflect.StringKind

	var fixKeys []protoreflect.MapKey
	m.Range(func(mk protoreflect.MapKey, mv protoreflect.Value) bool {
		if valIsMessage {
			sanitizeInvalidUTF8(mv.Message())
		}
		badVal := valIsString && !utf8.ValidString(mv.String())
		badKey := keyIsString && !utf8.ValidString(mk.String())
		if badVal || badKey {
			fixKeys = append(fixKeys, mk)
		}
		return true
	})
	for _, mk := range fixKeys {
		val := m.Get(mk)
		if valIsString && !utf8.ValidString(val.String()) {
			val = protoreflect.ValueOfString(strings.ToValidUTF8(val.String(), utf8Replacement))
		}
		if keyIsString && !utf8.ValidString(mk.String()) {
			m.Clear(mk)
			m.Set(protoreflect.ValueOfString(strings.ToValidUTF8(mk.String(), utf8Replacement)).MapKey(), val)
		} else {
			m.Set(mk, val)
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
