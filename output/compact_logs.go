package output

import (
	"context"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// UploadAndSendLogs - Filters the log file, then uploads it
func UploadAndSendLogs(ctx context.Context, server *state.Server, grant state.Grant, collectionOpts state.CollectionOpts, logger *util.Logger, logState state.TransientLogState) error {
	ls, r := transform.LogStateToLogSnapshot(server, logState)
	s := pganalyze_collector.CompactSnapshot{
		BaseRefs: &r,
		Data:     &pganalyze_collector.CompactSnapshot_LogSnapshot{LogSnapshot: &ls},
	}
	snapshotGrant := state.Grant{Valid: true, S3URL: grant.S3URL, S3Fields: grant.S3Fields}
	return uploadAndSubmitCompactSnapshot(ctx, s, snapshotGrant, server, collectionOpts, logger, logState.CollectedAt, false, "logs")
}
