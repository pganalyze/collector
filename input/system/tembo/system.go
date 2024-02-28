package tembo

import (
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState - Gets system information for a Tembo Cloud instance
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
	// TODO(ianstanton) Fetch system metrics from Tembo Cloud API
	return
}
