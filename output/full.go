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
	"strings"
	"time"

	"github.com/google/uuid"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SendFull(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, newState state.PersistedState, diffState state.DiffState, transientState state.TransientState, collectedIntervalSecs uint32) error {
	s := transform.StateToSnapshot(newState, diffState, transientState)
	s.CollectedIntervalSecs = collectedIntervalSecs
	err := verifyIntegrity(&s)
	if err != nil {
		logger.PrintError("Snapshot integrity check failed: %s; please contact support", err)
		// Don't return an error here, since that would skip the state update, and
		// if the integrity failure is due to a state diffing issue, we would not
		// be able to make progress. Instead, send a failed snapshot directly.
		return SendFailedFull(ctx, server, collectionOpts, logger)
	} else {
		return submitFull(ctx, s, server, collectionOpts, logger, newState.CollectedAt, false)
	}
}

func SendFailedFull(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger) error {
	s := snapshot.FullSnapshot{FailedRun: true}
	return submitFull(ctx, s, server, collectionOpts, logger, time.Now(), true)
}

func submitFull(ctx context.Context, s snapshot.FullSnapshot, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, collectedAt time.Time, quiet bool) error {
	var err error
	var data []byte

	snapshotUUID, err := uuid.NewV7()
	if err != nil {
		logger.PrintError("Error generating snapshot UUID: %s", err)
		return err
	}

	s.CollectorErrors = logger.ErrorMessages
	s.SnapshotVersionMajor = 1
	s.SnapshotVersionMinor = 0
	s.CollectorVersion = util.CollectorNameAndVersion
	s.SnapshotUuid = snapshotUUID.String()
	s.CollectedAt = timestamppb.New(collectedAt)
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
		if collectionOpts.OutputAsJson {
			debugOutputAsJSON(logger, compressedData)
		} else if !quiet {
			logger.PrintInfo("Collected snapshot successfully")
		}
		return nil
	}

	if server.WebSocket.Load() != nil {
		server.SnapshotStream <- compressedData.Bytes()
		logger.PrintInfo("Submitted full snapshot successfully")
		return nil
	}

	s3Location, err := uploadSnapshot(ctx, server.Config.HTTPClientWithRetry, *server.Grant.Load(), logger, compressedData, snapshotUUID.String())
	if err != nil {
		logger.PrintError("Error uploading snapshot: %s", err)
		return err
	}

	return submitSnapshot(ctx, server, collectionOpts, logger, s3Location, collectedAt, quiet)
}

func verifyIntegrity(s *snapshot.FullSnapshot) error {
	if len(s.DatabaseInformations) != len(s.DatabaseReferences) {
		return fmt.Errorf("found %d DatabaseInformations but %d DatabaseReferences", len(s.DatabaseInformations), len(s.DatabaseReferences))
	}
	if len(s.RoleInformations) != len(s.RoleReferences) {
		return fmt.Errorf("found %d RoleInformations but %d RoleReferences", len(s.RoleInformations), len(s.RoleReferences))
	}
	if len(s.TablespaceInformations) != len(s.TablespaceReferences) {
		return fmt.Errorf("found %d TablespaceInformations but %d TablespaceReferences", len(s.TablespaceInformations), len(s.TablespaceReferences))
	}
	if len(s.RelationInformations) != len(s.RelationReferences) {
		return fmt.Errorf("found %d RelationInformations but %d RelationReferences", len(s.RelationInformations), len(s.RelationReferences))
	}
	if len(s.IndexInformations) != len(s.IndexReferences) {
		return fmt.Errorf("found %d IndexInformations but %d IndexReferences", len(s.IndexInformations), len(s.IndexReferences))
	}
	if len(s.FunctionInformations) != len(s.FunctionReferences) {
		return fmt.Errorf("found %d FunctionInformations but %d FunctionReferences", len(s.FunctionInformations), len(s.FunctionReferences))
	}
	if len(s.QueryInformations) != len(s.QueryReferences) {
		return fmt.Errorf("found %d QueryInformations but %d QueryReferences", len(s.QueryInformations), len(s.QueryReferences))
	}

	return nil
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

	s := &snapshot.FullSnapshot{}
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

func submitSnapshot(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, s3Location string, collectedAt time.Time, quiet bool) error {
	requestURL := server.Config.APIBaseURL + "/v2/snapshots"

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

	msg, err := parseSnapshotResponse(resp, collectionOpts)
	if err != nil {
		return err
	}

	if len(msg) > 0 && collectionOpts.TestRun {
		logger.PrintInfo("  %s", msg)
	} else if !quiet {
		logger.PrintInfo("Submitted full snapshot successfully")
	}

	return nil
}
