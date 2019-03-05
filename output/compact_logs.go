package output

import (
	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// UploadAndSendLogs - Filters the log file, then uploads it to the storage and sends the metadata to the API
func UploadAndSendLogs(server state.Server, grant state.GrantLogs, collectionOpts state.CollectionOpts, logger *util.Logger, logState state.LogState) error {
	for _, logFile := range logState.LogFiles {
		logFile.FilterLogSecret = state.ParseFilterLogSecret(server.Config.FilterLogSecret)
	}

	if server.Config.FilterQuerySample == "all" {
		logState.QuerySamples = []state.PostgresQuerySample{}
	}

	if collectionOpts.SubmitCollectedData && grant.EncryptionKey.CiphertextBlob != "" {
		logState.LogFiles = EncryptAndUploadLogfiles(server.Config.HTTPClient, grant.Logdata, grant.EncryptionKey, logger, logState.LogFiles)
	}

	ls, r := transform.LogStateToLogSnapshot(logState)
	s := pganalyze_collector.CompactSnapshot{
		BaseRefs: &r,
		Data:     &pganalyze_collector.CompactSnapshot_LogSnapshot{LogSnapshot: &ls},
	}

	return uploadAndSubmitCompactSnapshot(s, grant.Snapshot, server, collectionOpts, logger, logState.CollectedAt, false, "logs")
}
