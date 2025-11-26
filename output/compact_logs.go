package output

import (
	"context"

	"github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/output/transform"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// UploadAndSendLogs - Filters the log file, then uploads it
func UploadAndSendLogs(ctx context.Context, server *state.Server, collectionOpts state.CollectionOpts, logger *util.Logger, logState state.TransientLogState) error {
	ls, r := transform.LogStateToLogSnapshot(server, logState)
	s := pganalyze_collector.CompactSnapshot{
		BaseRefs: &r,
		Data:     &pganalyze_collector.CompactSnapshot_LogSnapshot{LogSnapshot: &ls},
	}
	return uploadAndSubmitCompactSnapshot(ctx, s, server, collectionOpts, logger, logState.CollectedAt, false, "logs")
}
