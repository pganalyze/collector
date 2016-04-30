package logs

import (
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/explain"
	"github.com/pganalyze/collector/snapshot"
)

// GetLogLines - Retrieves all new log lines for this system and returns them
func GetLogLines(config config.DatabaseConfig) (lines []*snapshot.LogLine, explainInputs []explain.ExplainInput) {
	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceID != "" {
		lines, explainInputs = getFromAmazonRds(config)
	}

	return
}
