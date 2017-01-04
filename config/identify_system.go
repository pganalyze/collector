package config

import (
	"fmt"
	"os"
)

// Figure out if we're self-hosted or on RDS, as well as what ID we can use - Heroku is treated separately
func identifySystem(config ServerConfig) (systemType string, systemScope string, systemID string) {
	// Allow overrides from config or env variables
	systemType = config.SystemType
	systemScope = config.SystemScope
	systemID = config.SystemScope

	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceID != "" || systemType == "amazon_rds" {
		systemType = "amazon_rds"
		if systemScope == "" {
			systemScope = config.AwsRegion
		}
		if systemID == "" {
			systemID = config.AwsDbInstanceID
		}
	} else {
		systemType = "self_hosted"
		if systemID == "" {
			hostname := config.GetDbHost()
			if hostname == "" || hostname == "localhost" || hostname == "127.0.0.1" {
				hostname, _ = os.Hostname()
			}
			systemID = fmt.Sprintf("%s:%d/%s", hostname, config.GetDbPort(), config.GetDbName())
		}
	}
	return
}
