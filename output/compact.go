package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func uploadAndSubmitCompactSnapshot(ctx context.Context, s *pganalyze_collector.CompactSnapshot, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, collectedAt time.Time) error {
	var err error

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

	if !collectionOpts.SubmitCollectedData {
		if collectionOpts.OutputAsJson {
			debugCompactOutputAsJSON(logger, s)
		} else {
			logger.PrintInfo("Collected compact %s snapshot successfully", kindFromCompactSnapshot(s))
		}
		return nil
	}

	server.CompactSnapshotUpload <- s

	return nil
}

func kindFromCompactSnapshot(s *pganalyze_collector.CompactSnapshot) string {
	switch s.Data.(type) {
	case *pganalyze_collector.CompactSnapshot_ActivitySnapshot:
		return "activity"
	case *pganalyze_collector.CompactSnapshot_LogSnapshot:
		return "logs"
	case *pganalyze_collector.CompactSnapshot_SystemSnapshot:
		return "system"
	case *pganalyze_collector.CompactSnapshot_QueryRunSnapshot:
		return "query_run"
	}
	return "unknown"
}

func debugCompactOutputAsJSON(logger *util.Logger, s *pganalyze_collector.CompactSnapshot) {
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
