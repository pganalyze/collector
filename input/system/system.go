package system

import (
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/system/rds"
	"github.com/pganalyze/collector/input/system/selfhosted"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetLogLines - Retrieves all new log lines for this system and returns them
func GetLogLines(config config.ServerConfig) (lines []state.LogLine, explainInputs []state.PostgresExplainInput) {
	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceID != "" {
		// This is disabled for now, until we rework log processing
		//lines, explainInputs = rds.GetLogLines(config)
	}

	return
}

// GetSystemState - Retrieves a system snapshot for this system and returns it
func GetSystemState(config config.ServerConfig, logger *util.Logger, dataDirectory string) (system state.SystemState) {
	dbHost := config.GetDbHost()
	if config.AwsDbInstanceID != "" {
		// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
		system = rds.GetSystemState(config, logger, dataDirectory)
	} else if dbHost == "" || dbHost == "localhost" || dbHost == "127.0.0.1" {
		system = selfhosted.GetSystemState(config, logger, dataDirectory)
	}

	return
}
