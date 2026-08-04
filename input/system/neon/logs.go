package neon

import "github.com/pganalyze/collector/config"

// LogDatabaseFallback returns the database to attribute log lines to when Neon's
// log_line_prefix omits %d (it only ever serves the configured database).
// Returns "" for non-Neon servers.
func LogDatabaseFallback(config config.ServerConfig) string {
	if config.SystemType != "neon" {
		return ""
	}
	return config.GetDbName()
}
