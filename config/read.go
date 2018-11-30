package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/go-ini/ini"

	"github.com/pganalyze/collector/util"
)

func getDefaultConfig() *ServerConfig {
	config := &ServerConfig{
		APIBaseURL:              "https://api.pganalyze.com",
		AwsRegion:               "us-east-1",
		SectionName:             "default",
		QueryStatsInterval:      60,
		MaxCollectorConnections: 10,
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
	if enableLogs := os.Getenv("PGA_ENABLE_LOGS"); enableLogs != "" && enableLogs != "0" {
		config.EnableLogs = true
	}
	if enableReports := os.Getenv("PGA_ENABLE_REPORTS"); enableReports != "" && enableReports != "0" {
		config.EnableReports = true
	}
	if enableActivity := os.Getenv("PGA_ENABLE_ACTIVITY"); enableActivity != "" && enableActivity != "0" {
		config.EnableActivity = true
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
	if dbSslMode := os.Getenv("DB_SSLMODE"); dbSslMode != "" {
		config.DbSslMode = dbSslMode
	}
	if dbSslRootCert := os.Getenv("DB_SSLROOTCERT"); dbSslRootCert != "" {
		config.DbSslRootCert = dbSslRootCert
	}
	if dbSslRootCertContents := os.Getenv("DB_SSLROOTCERT_CONTENTS"); dbSslRootCertContents != "" {
		config.DbSslRootCertContents = dbSslRootCertContents
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
	if logLocation := os.Getenv("LOG_LOCATION"); logLocation != "" {
		config.LogLocation = logLocation
		config.EnableLogs = true
	}
	// Note: We don't support LogDockerTail here since it would require the "docker"
	// binary inside the pganalyze container (as well as full Docker access), instead
	// the approach for using pganalyze as a sidecar container alongside Postgres
	// currently requires writing to a file and then mounting that as a volume
	// inside the pganalyze container.
	if ignoreTablePattern := os.Getenv("IGNORE_TABLE_PATTERN"); ignoreTablePattern != "" {
		config.IgnoreTablePattern = ignoreTablePattern
	}
	if queryStatsInterval := os.Getenv("QUERY_STATS_INTERVAL"); queryStatsInterval != "" {
		config.QueryStatsInterval, _ = strconv.Atoi(queryStatsInterval)
	}
	if maxCollectorConnections := os.Getenv("MAX_COLLECTOR_CONNECTION"); maxCollectorConnections != "" {
		config.MaxCollectorConnections, _ = strconv.Atoi(maxCollectorConnections)
	}

	return config
}

// Read - Reads the configuration from the specified filename, or fall back to the default config
func Read(logger *util.Logger, filename string) (Config, error) {
	var conf Config
	var err error

	if _, err = os.Stat(filename); err == nil {
		configFile, err := ini.Load(filename)
		if err != nil {
			return conf, err
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
				return conf, err
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

			if config.DbSslRootCertContents != "" {
				sslRootTmpFile, err := ioutil.TempFile("", "")
				if err != nil {
					return conf, err
				}
				_, err = sslRootTmpFile.WriteString(config.DbSslRootCertContents)
				if err != nil {
					return conf, err
				}
				err = sslRootTmpFile.Close()
				if err != nil {
					return conf, err
				}
				config.DbSslRootCert = sslRootTmpFile.Name()
			}

			config.SectionName = section.Name()
			config.SystemType, config.SystemScope, config.SystemID = identifySystem(*config)

			config.Identifier = ServerIdentifier{
				APIKey:      config.APIKey,
				APIBaseURL:  config.APIBaseURL,
				SystemID:    config.SystemID,
				SystemType:  config.SystemType,
				SystemScope: config.SystemScope,
			}

			if config.GetDbName() != "" {
				// Ensure we have no duplicate identifiers within one collector
				skip := false
				for _, server := range conf.Servers {
					if config.Identifier == server.Identifier {
						skip = true
					}
				}
				if skip {
					logger.PrintError("Skipping config section %s, detected as duplicate", config.SectionName)
				} else {
					conf.Servers = append(conf.Servers, *config)
				}
			}
		}

		if len(conf.Servers) == 0 {
			return conf, fmt.Errorf("Configuration file is empty, please edit %s and reload the collector", filename)
		}
	} else {
		if os.Getenv("DYNO") != "" && os.Getenv("PORT") != "" {
			conf = handleHeroku()
		} else if os.Getenv("PGA_API_KEY") != "" {
			config := getDefaultConfig()
			config.SystemType, config.SystemScope, config.SystemID = identifySystem(*config)
			conf.Servers = append(conf.Servers, *config)
		} else {
			return conf, fmt.Errorf("No configuration file found at %s, and no environment variables set", filename)
		}
	}

	return conf, nil
}
