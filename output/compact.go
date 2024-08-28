package output

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func uploadAndSubmitCompactSnapshot(ctx context.Context, s pganalyze_collector.CompactSnapshot, grant state.Grant, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, collectedAt time.Time, quiet bool, kind string) error {
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

	if server.WebSocket.Load() != nil {
		server.SnapshotStream <- compressedData.Bytes()
		logger.PrintVerbose("Submitted compact %s snapshot successfully", kind)
		return nil
	}

	s3Location, err := uploadSnapshot(ctx, server.Config.HTTPClientWithRetry, grant, logger, compressedData, snapshotUUID.String())
	if err != nil {
		logger.PrintError("Error uploading snapshot: %s", err)
		return err
	}

	return submitCompactSnapshot(ctx, server, collectionOpts, logger, s3Location, collectedAt, quiet, kind)
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

func submitCompactSnapshot(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, s3Location string, collectedAt time.Time, quiet bool, kind string) error {
	requestURL := server.Config.APIBaseURL + "/v2/snapshots/compact"

	if collectionOpts.TestRun {
		requestURL = server.Config.APIBaseURL + "/v2/snapshots/test"
	}

	data := url.Values{
		"s3_location":  {s3Location},
		"collected_at": {fmt.Sprintf("%d", collectedAt.Unix())},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Pganalyze-Api-Key", server.Config.APIKey)
	req.Header.Set("Pganalyze-System-Id", server.Config.SystemID)
	req.Header.Set("Pganalyze-System-Type", server.Config.SystemType)
	req.Header.Set("Pganalyze-System-Scope", server.Config.SystemScope)
	req.Header.Set("Pganalyze-System-Id-Fallback", server.Config.SystemIDFallback)
	req.Header.Set("Pganalyze-System-Type-Fallback", server.Config.SystemTypeFallback)
	req.Header.Set("Pganalyze-System-Scope-Fallback", server.Config.SystemScopeFallback)
	req.Header.Set("User-Agent", util.CollectorNameAndVersion)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json,text/plain")

	resp, err := server.Config.HTTPClientWithRetry.Do(req)
	if err != nil {
		return util.CleanHTTPError(err)
	}

	msg, serverURL, err := parseSnapshotResponse(resp, collectionOpts)
	if err != nil {
		return err
	}

	if serverURL != "" {
		server.PGAnalyzeURL = serverURL
	}

	if len(msg) > 0 && collectionOpts.TestRun {
		logger.PrintInfo("  %s", msg)
	} else if !quiet {
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

	return nil
}
