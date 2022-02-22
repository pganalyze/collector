package system

import (
	"os"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/system/crunchy_bridge"
	"github.com/pganalyze/collector/input/system/rds"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// DownloadLogFiles - Downloads all new log files for the remote system and returns them
func DownloadLogFiles(server *state.Server, globalCollectionOpts state.CollectionOpts, logger *util.Logger) (psl state.PersistedLogState, files []state.LogFile, querySamples []state.PostgresQuerySample, err error) {
	if server.Config.SystemType == "amazon_rds" {
		psl, files, querySamples, err = rds.DownloadLogFiles(server.LogPrevState, server.Config, logger)
		if err != nil {
			return
		}
	} else if server.Config.SystemType == "crunchy_bridge" {
		psl, files, querySamples, err = crunchy_bridge.DownloadLogFiles(server, globalCollectionOpts, logger)
		if err != nil {
			return
		}
	} else {
		psl = server.LogPrevState
	}

	return
}

// GetSystemState - Retrieves a system snapshot for this system and returns it
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
	dbHost := config.GetDbHost()
	if config.SystemType == "amazon_rds" {
		system = rds.GetSystemState(config, logger)
	} else if config.SystemType == "google_cloudsql" {
		system.Info.Type = state.GoogleCloudSQLSystem
	} else if config.SystemType == "azure_database" {
		system.Info.Type = state.AzureDatabaseSystem
	} else if config.SystemType == "heroku" {
		system.Info.Type = state.HerokuSystem
	} else if config.SystemType == "crunchy_bridge" {
		system.Info.Type = state.CrunchyBridgeSystem
	} else if dbHost == "" || dbHost == "localhost" || dbHost == "127.0.0.1" || os.Getenv("PGA_ALWAYS_COLLECT_SYSTEM_DATA") != "" {
		system = selfhosted.GetSystemState(config, logger)
	}

	system.Info.SystemID = config.SystemID
	system.Info.SystemScope = config.SystemScope

	return
}
