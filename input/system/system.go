package system

import (
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/input/system/rds"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetLogLines - Retrieves all new log lines for this system and returns them
func GetLogLines(config config.DatabaseConfig) (lines []state.LogLine, explainInputs []state.PostgresExplainInput) {
	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceID != "" {
		lines, explainInputs = rds.GetLogLines(config)
	}

	return
}

// GetSystemState - Retrieves a system snapshot for this system and returns it
func GetSystemState(config config.DatabaseConfig, logger *util.Logger) (system state.SystemState) {
	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceID != "" {
		system = rds.GetSystemState(config, logger)
	}

	return
}
