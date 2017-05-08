package output

import (
	"bytes"
	"compress/zlib"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/satori/go.uuid"
)

func UploadAndSendLogs(server state.Server, grant state.GrantLogs, collectionOpts state.CollectionOpts, logger *util.Logger, logState state.LogState) error {
	if collectionOpts.SubmitCollectedData && grant.EncryptionKey.CiphertextBlob != "" {
		logState.LogFiles = EncryptAndUploadLogfiles(grant.Logdata, grant.EncryptionKey, logger, logState.LogFiles)
	}

	ls, r := transform.LogStateToLogSnapshot(logState)

	return submitLogs(ls, r, grant.Snapshot, server, collectionOpts, logger, logState.CollectedAt, false)
}

func submitLogs(ls pganalyze_collector.CompactLogSnapshot, r pganalyze_collector.CompactSnapshot_BaseRefs, s3 state.GrantS3, server state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, collectedAt time.Time, quiet bool) error {
	var err error
	var data []byte

	snapshotUUID := uuid.NewV4()

	s := pganalyze_collector.CompactSnapshot{}
	s.SnapshotVersionMajor = 1
	s.SnapshotVersionMinor = 0
	s.CollectorVersion = util.CollectorNameAndVersion
	s.SnapshotUuid = snapshotUUID.String()
	s.CollectedAt, _ = ptypes.TimestampProto(collectedAt)
	s.BaseRefs = &r
	s.Data = &pganalyze_collector.CompactSnapshot_LogSnapshot{LogSnapshot: &ls}

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
		debugCompactOutputAsJSON(logger, compressedData)
		return nil
	}

	s3Location, err := uploadCompactSnapshot(s3, logger, compressedData, snapshotUUID.String())
	if err != nil {
		logger.PrintError("Error uploading to S3: %s", err)
		return err
	}

	return submitCompactSnapshot(server, collectionOpts, logger, s3Location, collectedAt, quiet)
}
