package input

import (
	"database/sql"
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// DownloadLogs - Downloads a "logs" snapshot of log data we need on a regular interval
func DownloadLogs(server state.Server, connection *sql.DB, collectionOpts state.CollectionOpts, logger *util.Logger) (ls state.LogState, err error) {
	var querySamples []state.PostgresQuerySample

	ls.CollectedAt = time.Now()
	ls.LogFiles, querySamples = system.DownloadLogFiles(server.Config, logger)

	if false && collectionOpts.CollectExplain && server.Grant.Config.Features.Explain {
		ls.QuerySamples = postgres.RunExplain(connection, querySamples)
	} else {
		ls.QuerySamples = querySamples
	}
	return
}
