package config

import (
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

			if config.DbName != "" {
				servers = append(servers, *config)
			}
		}

		if len(servers) == 0 {
			logger.PrintError("Error: Configuration is empty, please edit %s and reload the collector", filename)
		}
	} else {
		if os.Getenv("DYNO") != "" && os.Getenv("PORT") != "" {
			servers = handleHeroku()
		} else {
			servers = append(servers, *getDefaultConfig())
		}
	}

	return servers, nil
}
