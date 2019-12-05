package system

import (
	"os"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/system/rds"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// DownloadLogFiles - Downloads all new log files for the remote system and returns them
func DownloadLogFiles(config config.ServerConfig, logger *util.Logger) (files []state.LogFile, querySamples []state.PostgresQuerySample) {
	if config.SystemType == "amazon_rds" {
		files, querySamples = rds.DownloadLogFiles(config, logger)
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
	} else if dbHost == "" || dbHost == "localhost" || dbHost == "127.0.0.1" || os.Getenv("PGA_ALWAYS_COLLECT_SYSTEM_DATA") != "" {
		system = selfhosted.GetSystemState(config, logger)
	}

	system.Info.SystemID = config.SystemID
	system.Info.SystemScope = config.SystemScope

	return
}
