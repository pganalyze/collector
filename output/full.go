package output

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	uuid "github.com/satori/go.uuid"
)

func SendFull(server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState, collectedIntervalSecs uint32) error {
	s := transform.StateToSnapshot(newState, diffState, transientState)
	s.CollectedIntervalSecs = collectedIntervalSecs
	s.CollectorErrors = logger.ErrorMessages

	return submitFull(s, server, collectionOpts, logger, newState.CollectedAt, false)
}

func SendFailedFull(server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) error {
	s := snapshot.FullSnapshot{FailedRun: true, CollectorErrors: logger.ErrorMessages}
	return submitFull(s, server, collectionOpts, logger, time.Now(), true)
}

func submitFull(s snapshot.FullSnapshot, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, collectedAt time.Time, quiet bool) error {
	var err error
	var data []byte

	snapshotUUID := uuid.NewV4()

	s.SnapshotVersionMajor = 1
	s.SnapshotVersionMinor = 0
	s.CollectorVersion = util.CollectorNameAndVersion
	s.SnapshotUuid = snapshotUUID.String()
	s.CollectedAt, _ = ptypes.TimestampProto(collectedAt)
	s.CollectorLogSnapshotDisabled = server.CollectionStatus.LogSnapshotDisabled
	s.CollectorLogSnapshotDisabledReason = server.CollectionStatus.LogSnapshotDisabledReason

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
		debugOutputAsJSON(logger, compressedData)
		return nil
	}

	s3Location, err := uploadSnapshot(server.Config.HTTPClientWithRetry, server.Grant, logger, compressedData, snapshotUUID.String())
	if err != nil {
		logger.PrintError("Error uploading to S3: %s", err)
		return err
	}

	return submitSnapshot(server, collectionOpts, logger, s3Location, collectedAt, quiet)
}

func debugOutputAsJSON(logger *util.Logger, compressedData bytes.Buffer) {
	var err error
	var data bytes.Buffer

	r, err := zlib.NewReader(&compressedData)
	if err != nil {
		logger.PrintError("Failed to decompress protocol buffers: %s", err)
		return
	}
	defer r.Close()

	io.Copy(&data, r)

	s := &pganalyze_collector.FullSnapshot{}
	if err = proto.Unmarshal(data.Bytes(), s); err != nil {
		logger.PrintError("Failed to re-read protocol buffers: %s", err)
		return
	}

	var out bytes.Buffer
	var marshaler jsonpb.Marshaler
	dataJSON, err := marshaler.MarshalToString(s)
	if err != nil {
		logger.PrintError("Failed to transform protocol buffers to JSON: %s", err)
		return
	}
	json.Indent(&out, []byte(dataJSON), "", "\t")
	logger.PrintInfo("Dry run - data that would have been sent will be output on stdout:\n")
	fmt.Printf("%s\n", out.String())
}

func submitSnapshot(server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, s3Location string, collectedAt time.Time, quiet bool) error {
	requestURL := server.Config.APIBaseURL + "/v2/snapshots"

	if collectionOpts.TestRun {
		requestURL = server.Config.APIBaseURL + "/v2/snapshots/test"
	}

	data := url.Values{
		"s3_location":  {s3Location},
		"collected_at": {fmt.Sprintf("%d", collectedAt.Unix())},
	}

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(data.Encode()))
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
		logger.PrintInfo("Submitted snapshot successfully")
	}

	return nil
}
