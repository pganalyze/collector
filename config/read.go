package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-ini/ini"

	"github.com/pganalyze/collector/util"
)

func getDefaultConfig() *ServerConfig {
	config := &ServerConfig{
		APIBaseURL:  "https://api.pganalyze.com",
		DbHost:      "localhost",
		DbPort:      5432,
		DbSslMode:   "prefer",
		AwsRegion:   "us-east-1",
		SectionName: "default",
	}

	// The environment variables are the default way to configure when running inside a Docker container.
	if apiKey := os.Getenv("PGA_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if apiBaseURL := os.Getenv("PGA_API_BASEURL"); apiBaseURL != "" {
		config.APIBaseURL = apiBaseURL
	}
	if systemID := os.Getenv("PGA_API_SYSTEM_ID"); systemID != "" {
		config.SystemID = systemID
	}
	if systemType := os.Getenv("PGA_API_SYSTEM_TYPE"); systemType != "" {
		config.SystemType = systemType
	}
	if systemScope := os.Getenv("PGA_API_SYSTEM_SCOPE"); systemScope != "" {
		config.SystemScope = systemScope
	}
	if dbURL := os.Getenv("DB_URL"); dbURL != "" {
		config.DbURL = dbURL
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		config.DbName = dbName
	}
	if dbAllNames := os.Getenv("DB_ALL_NAMES"); dbAllNames == "1" {
		config.DbAllNames = true
	}
	if dbUsername := os.Getenv("DB_USERNAME"); dbUsername != "" {
		config.DbUsername = dbUsername
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		config.DbPassword = dbPassword
	}
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.DbHost = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		config.DbPort, _ = strconv.Atoi(dbPort)
	}
	if dbStatementFrequency := os.Getenv("DB_STATEMENT_FREQUENCY"); dbStatementFrequency != "" {
		config.DbStatementFrequency, _ = strconv.Atoi(dbStatementFrequency)
	} else {
		config.DbStatementFrequency = 1
	}
	if awsRegion := os.Getenv("AWS_REGION"); awsRegion != "" {
		config.AwsRegion = awsRegion
	}
	if awsInstanceID := os.Getenv("AWS_INSTANCE_ID"); awsInstanceID != "" {
		config.AwsDbInstanceID = awsInstanceID
	}
	if awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID"); awsAccessKeyID != "" {
		config.AwsAccessKeyID = awsAccessKeyID
	}
	if awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); awsSecretAccessKey != "" {
		config.AwsSecretAccessKey = awsSecretAccessKey
	}

	return config
}

// Read - Reads the configuration from the specified filename, or fall back to the default config
func Read(logger *util.Logger, filename string) ([]ServerConfig, error) {
	var servers []ServerConfig
	var err error

	if _, err = os.Stat(filename); err == nil {
		configFile, err := ini.Load(filename)
		if err != nil {
			return servers, err
		}

		defaultConfig := getDefaultConfig()

		err = configFile.Section("pganalyze").MapTo(defaultConfig)
		if err != nil {
			logger.PrintVerbose("Failed to map pganalyze section: %s", err)
		}

		sections := configFile.Sections()
		for _, section := range sections {
			config := &ServerConfig{}
			*config = *defaultConfig

			err = section.MapTo(config)
			if err != nil {
				return servers, err
			}

			dbNameParts := []string{}
			for _, s := range strings.Split(config.DbName, ",") {
				dbNameParts = append(dbNameParts, strings.TrimSpace(s))
			}
			config.DbName = dbNameParts[0]
			if len(dbNameParts) == 2 && dbNameParts[1] == "*" {
				config.DbAllNames = true
			} else {
				config.DbExtraNames = dbNameParts[1:]
			}

			config.SectionName = section.Name()
			config.SystemType, config.SystemScope, config.SystemID = identifySystem(*config)

			if config.GetDbName() != "" {
				// Ensure we have no duplicate System Type+Scope+ID within one collector
				skip := false
				for _, server := range servers {
					if config.SystemType == server.SystemType &&
						config.SystemScope == server.SystemScope &&
						config.SystemID == server.SystemID {
						skip = true
					}
				}
				if skip {
					logger.PrintError("Skipping config section %s, detected as duplicate", config.SectionName)
				} else {
					servers = append(servers, *config)
				}
			}
		}

		if len(servers) == 0 {
			return servers, fmt.Errorf("Configuration file is empty, please edit %s and reload the collector", filename)
		}
	} else {
		if os.Getenv("DYNO") != "" && os.Getenv("PORT") != "" {
			servers = handleHeroku()
		} else if os.Getenv("PGA_API_KEY") != "" {
			config := getDefaultConfig()
			config.SystemType, config.SystemScope, config.SystemID = identifySystem(*config)
			servers = append(servers, *config)
		} else {
			return servers, fmt.Errorf("No configuration file found at %s, and no environment variables set", filename)
		}
	}

	return servers, nil
}
