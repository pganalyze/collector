package tembo

import (
	"context"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// DownloadLogFiles - Gets log files for a Tembo instance
func DownloadLogFiles(ctx context.Context, server *state.Server, logger *util.Logger) (state.PersistedLogState, []state.LogFile, []state.PostgresQuerySample, error) {
	//TODO(ianstanton) - Implement tembo log file download for tembo
	return server.LogPrevState, nil, nil, nil
}
