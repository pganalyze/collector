package config

import (
	"fmt"
	"os"
)

// Figure out if we're self-hosted or on RDS, as well as what ID we can use - Heroku is treated separately
func identifySystem(config ServerConfig) (systemType string, systemScope string, systemScopeFallback string, systemID string) {
	// Allow overrides from config or env variables
	systemType = config.SystemType
	systemScope = config.SystemScope
	systemScopeFallback = config.SystemScopeFallback
	systemID = config.SystemID

	if config.AwsDbInstanceID != "" || systemType == "amazon_rds" {
		systemType = "amazon_rds"
		if systemScope == "" {
			if config.AwsAccountID != "" {
				systemScope = config.AwsRegion + "/" + config.AwsAccountID
				if systemScopeFallback == "" {
					systemScopeFallback = config.AwsRegion
				}
			} else {
				systemScope = config.AwsRegion
			}
		}
		if systemID == "" {
			systemID = config.AwsDbInstanceID
		}
	} else if config.AzureDbServerName != "" || systemType == "azure_database" {
		systemType = "azure_database"
		if systemID == "" {
			systemID = config.AzureDbServerName
		}
	} else if (config.GcpProjectID != "" && config.GcpCloudSQLInstanceID != "") || systemType == "google_cloudsql" {
		systemType = "google_cloudsql"
		if systemScope == "" {
			systemScope = config.GcpProjectID
		}
		if systemID == "" {
			systemID = config.GcpCloudSQLInstanceID
		}
	} else if (config.CrunchyBridgeClusterID != "") || systemType == "crunchy_bridge" {
		systemType = "crunchy_bridge"
		if systemID == "" {
			systemID = config.CrunchyBridgeClusterID
		}
	} else if (config.AivenServiceID != "") || systemType == "aiven" {
		systemType = "aiven"
		if systemID == "" {
			systemID = config.AivenProjectID + "-" + config.AivenServiceID
		}
	} else {
		systemType = "self_hosted"
		if systemID == "" {
			hostname := config.GetDbHost()
			if hostname == "" || hostname == "localhost" || hostname == "127.0.0.1" {
				hostname, _ = os.Hostname()
			}
			systemID = hostname
			if systemScope == "" {
				systemScope = fmt.Sprintf("%d/%s", config.GetDbPort(), config.GetDbName())
				if config.DbAllNames {
					systemScope += "*"
				}
			}
		}
	}
	return
}
