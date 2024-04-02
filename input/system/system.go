package system

import (
	"context"

	"github.com/pganalyze/collector/input/system/tembo"

	"github.com/pganalyze/collector/input/postgres"
	"github.com/pganalyze/collector/input/system/crunchy_bridge"
	"github.com/pganalyze/collector/input/system/rds"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// DownloadLogFiles - Downloads all new log files for the remote system and returns them
func DownloadLogFiles(ctx context.Context, server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (psl state.PersistedLogState, files []state.LogFile, querySamples []state.PostgresQuerySample, err error) {
	if server.Config.SystemType == "amazon_rds" {
		psl, files, querySamples, err = rds.DownloadLogFiles(ctx, server, logger)
		if err != nil {
			return
		}
	} else if server.Config.LogPgReadFile {
		psl, files, querySamples, err = postgres.LogPgReadFile(ctx, server, globalCollectionOpts, logger)
		if err != nil {
			return
		}
	} else {
		psl = server.LogPrevState
	}

	return
}

// GetSystemState - Retrieves a system snapshot for this system and returns it
func GetSystemState(server *state.Server, logger *util.Logger, globalCollectionOpts state.CollectionOpts) (system state.SystemState) {
	config := server.Config
	dbHost := config.GetDbHost()
	if config.SystemType == "amazon_rds" {
		system = rds.GetSystemState(server, logger)
	} else if config.SystemType == "google_cloudsql" {
		system.Info.Type = state.GoogleCloudSQLSystem
		server.SelfTest.MarkCollectionAspectNotAvailable(state.CollectionAspectSystemStats, "not available on this platform")
	} else if config.SystemType == "azure_database" {
		system.Info.Type = state.AzureDatabaseSystem
		server.SelfTest.MarkCollectionAspectNotAvailable(state.CollectionAspectSystemStats, "not available on this platform")
	} else if config.SystemType == "heroku" {
		system.Info.Type = state.HerokuSystem
		server.SelfTest.MarkCollectionAspectNotAvailable(state.CollectionAspectSystemStats, "not available on this platform")
	} else if config.SystemType == "crunchy_bridge" {
		system = crunchy_bridge.GetSystemState(server, logger)
	} else if config.SystemType == "aiven" {
		system.Info.Type = state.AivenSystem
		server.SelfTest.MarkCollectionAspectNotAvailable(state.CollectionAspectSystemStats, "not available on this platform")
	} else if config.SystemType == "tembo" {
		system = tembo.GetSystemState(server, logger)
	} else if dbHost == "" || dbHost == "localhost" || dbHost == "127.0.0.1" || config.AlwaysCollectSystemData {
		system = selfhosted.GetSystemState(server, logger)
	} else {
		// system state is _available_ here, just not
		if globalCollectionOpts.TestRun {
			// Detected as self hosted, but not collecting system state as we
			// didn't detect the collector is running on the same instance as
			// the database server.
			// Leave logs for if this is a test run.
			server.SelfTest.MarkCollectionAspectNotAvailable(state.CollectionAspectSystemStats, "remote host (%s) was specified for the database address", dbHost)
			server.SelfTest.HintCollectionAspect(state.CollectionAspectSystemStats, "Consider enabling always_collect_system_data if the database is running on the same system as the collector")
			logger.PrintInfo("Skipping collection of system state: remote host (%s) was specified for the database address. Consider enabling always_collect_system_data if the database is running on the same system as the collector", dbHost)
		}
	}

	system.Info.SystemID = config.SystemID
	system.Info.SystemScope = config.SystemScope

	return
}
