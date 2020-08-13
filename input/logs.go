package input

import (
	"time"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// DownloadLogs - Downloads a "logs" snapshot of log data we need on a regular interval
func DownloadLogs(server state.Server, prevLogState state.PersistedLogState, collectionOpts state.CollectionOpts, logger *util.Logger) (tls state.TransientLogState, pls state.PersistedLogState, err error) {
	var querySamples []state.PostgresQuerySample

	tls.CollectedAt = time.Now()
	pls, tls.LogFiles, querySamples = system.DownloadLogFiles(prevLogState, server.Config, logger)

	if server.Config.EnableLogExplain {
		tls.QuerySamples = postgres.RunExplain(server, querySamples, collectionOpts, logger)
	} else {
		tls.QuerySamples = querySamples
	}
	return
}
