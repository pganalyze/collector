package config

import (
	"os"
	"strconv"

	"github.com/go-ini/ini"
)

func GetDefaultConfig() *DatabaseConfig {
	config := &DatabaseConfig{
		APIURL:    "https://api.pganalyze.com/v1/snapshots",
		DbHost:    "localhost",
		DbPort:    5432,
		AwsRegion: "us-east-1",
	}

	// The environment variables are the default way to configure when running inside a Docker container.
	if apiKey := os.Getenv("PGA_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if apiURL := os.Getenv("PGA_API_URL"); apiURL != "" {
		config.APIURL = apiURL
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
	if awsInstanceId := os.Getenv("AWS_INSTANCE_ID"); awsInstanceId != "" {
		config.AwsDbInstanceId = awsInstanceId
	}
	if awsAccessKeyId := os.Getenv("AWS_ACCESS_KEY_ID"); awsAccessKeyId != "" {
		config.AwsAccessKeyId = awsAccessKeyId
	}
	if awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); awsSecretAccessKey != "" {
		config.AwsSecretAccessKey = awsSecretAccessKey
	}

	return config
}

func Read(filename string) ([]DatabaseConfig, error) {
	var databases []DatabaseConfig

	if _, err := os.Stat(filename); err == nil {
		configFile, err := ini.Load(filename)
		if err != nil {
			return databases, err
		}

		sections := configFile.Sections()
		for _, section := range sections {
			config := GetDefaultConfig()

			err = section.MapTo(config)
			if err != nil {
				return databases, err
			}

			config.SectionName = section.Name()

			if config.DbName != "" {
				databases = append(databases, *config)
			}
		}
	} else {
		databases = append(databases, *GetDefaultConfig())
	}

	return databases, nil
}
