package config

import (
	"fmt"
	"os"
	"strconv"

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

	if _, err := os.Stat(filename); err == nil {
		configFile, err := ini.Load(filename)
		if err != nil {
			return servers, err
		}

		sections := configFile.Sections()
		for _, section := range sections {
			config := getDefaultConfig()

			err = section.MapTo(config)
			if err != nil {
				return servers, err
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
