package output

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func uploadAndSubmitCompactSnapshot(ctx context.Context, s pganalyze_collector.CompactSnapshot, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, collectedAt time.Time, quiet bool, kind string) error {
	var err error
	var data []byte

	snapshotUUID, err := uuid.NewV7()
	if err != nil {
		logger.PrintError("Error generating snapshot UUID: %s", err)
		return err
	}

	s.SnapshotVersionMajor = 1
	s.SnapshotVersionMinor = 0
	s.CollectorVersion = util.CollectorNameAndVersion
	s.SnapshotUuid = snapshotUUID.String()
	s.CollectedAt = timestamppb.New(collectedAt)

	data, err = proto.Marshal(&s)
	if err != nil {
		logger.PrintError("Error marshaling protocol buffers")
		return err
	}

	var compressedData bytes.Buffer
	w := zlib.NewWriter(&compressedData)
	w.Write(data)
	w.Close()

	if !collectionOpts.SubmitCollectedData {
		if collectionOpts.OutputAsJson {
			debugCompactOutputAsJSON(logger, compressedData)
		} else if !quiet {
			logger.PrintInfo("Collected compact %s snapshot successfully", kind)
		}
		return nil
	}

	server.SnapshotStream <- state.Snapshot{Data: compressedData.Bytes(), SnapshotUuid: snapshotUUID.String(), CollectedAt: collectedAt, CompactSnapshot: true}

	if !collectionOpts.TestRun && !quiet {
		logger.PrintVerbose("Submitted compact %s snapshot successfully", kind)
		if server.CompactLogTime.IsZero() {
			server.CompactLogTime = time.Now().Truncate(time.Minute)
			server.CompactLogStats = make(map[string]uint8)
		} else {
			server.CompactLogStats[kind] = server.CompactLogStats[kind] + 1
			if time.Since(server.CompactLogTime) > time.Minute {
				var keys []string
				for k := range server.CompactLogStats {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				details := ""
				for i, kind := range keys {
					details += fmt.Sprintf("%d %s", server.CompactLogStats[kind], kind)
					if i < len(keys)-1 {
						details += ", "
					}
				}
				if len(details) > 0 {
					logger.PrintInfo("Submitted compact snapshots successfully: " + details)
				}
				server.CompactLogTime = time.Now().Truncate(time.Minute)
				server.CompactLogStats = make(map[string]uint8)
			}
		}
	}

	// TODO: This previously returned errors when not sending through websockets
	return nil
}

func debugCompactOutputAsJSON(logger *util.Logger, compressedData bytes.Buffer) {
	var err error
	var data bytes.Buffer

	r, err := zlib.NewReader(&compressedData)
	if err != nil {
		logger.PrintError("Failed to decompress protocol buffers: %s", err)
		return
	}
	defer r.Close()

	io.Copy(&data, r)

	s := &pganalyze_collector.CompactSnapshot{}
	if err = proto.Unmarshal(data.Bytes(), s); err != nil {
		logger.PrintError("Failed to re-read protocol buffers: %s", err)
		return
	}

	var out bytes.Buffer
	dataJSON, err := protojson.Marshal(s)
	if err != nil {
		logger.PrintError("Failed to transform protocol buffers to JSON: %s", err)
		return
	}
	json.Indent(&out, dataJSON, "", "\t")
	logger.PrintInfo("Dry run - data that would have been sent will be output on stdout:\n")
	fmt.Printf("%s\n", out.String())
}
