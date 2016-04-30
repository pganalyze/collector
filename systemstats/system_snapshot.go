package systemstats

import (
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/snapshot"
)

// GetSystemSnapshot - Retrieves a system snapshot for this system and returns it
func GetSystemSnapshot(config config.DatabaseConfig) (system *snapshot.System) {
	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceID != "" {
		system = getFromAmazonRds(config)
	}

	return
}
