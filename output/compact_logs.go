package output

import (
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

func UploadAndSendLogs(server state.Server, grant state.GrantLogs, collectionOpts state.CollectionOpts, logger *util.Logger, logState state.LogState) error {
	if collectionOpts.SubmitCollectedData && grant.EncryptionKey.CiphertextBlob != "" {
		logState.LogFiles = EncryptAndUploadLogfiles(grant.Logdata, grant.EncryptionKey, logger, logState.LogFiles)
	}

	ls, r := transform.LogStateToLogSnapshot(logState)
	s := pganalyze_collector.CompactSnapshot{
		BaseRefs: &r,
		Data:     &pganalyze_collector.CompactSnapshot_LogSnapshot{LogSnapshot: &ls},
	}

	return uploadAndSubmitCompactSnapshot(s, grant.Snapshot, server, collectionOpts, logger, logState.CollectedAt, false)
}
