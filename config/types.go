package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DatabaseConfig -
//   Contains the information how to connect to a single Postgres
//   database (on a given instance), with optional AWS credentials to get metrics
//   from AWS CloudWatch as well as RDS logfiles
type DatabaseConfig struct {
	APIKey     string `ini:"api_key"`
	APIURL     string `ini:"api_url"`
	DbURL      string `ini:"db_url"`
	DbName     string `ini:"db_name"`
	DbUsername string `ini:"db_username"`
	DbPassword string `ini:"db_password"`
	DbHost     string `ini:"db_host"`
	DbPort     int    `ini:"db_port"`
	DbSslMode  string `ini:"db_sslmode"`

	AwsRegion          string `ini:"aws_region"`
	AwsDbInstanceID    string `ini:"aws_db_instance_id"`
	AwsAccessKeyID     string `ini:"aws_access_key_id"`
	AwsSecretAccessKey string `ini:"aws_secret_access_key"`

	SectionName string
}

// GetPqOpenString - Gets the database configuration as a string that can be passed to lib/pq for connecting
func (config DatabaseConfig) GetPqOpenString() string {
	if config.DbURL != "" {
		return config.DbURL
	}

	dbinfo := fmt.Sprintf("user=%s dbname=%s host=%s port=%d connect_timeout=10 sslmode=%s",
		config.DbUsername, config.DbName, config.DbHost, config.DbPort, config.DbSslMode)

	if config.DbPassword != "" {
		dbinfo += fmt.Sprintf(" password=%s", config.DbPassword)
	}

	return dbinfo
}

// GetDbHost - Gets the database hostname from the given configuration
func (config DatabaseConfig) GetDbHost() string {
	if config.DbURL != "" {
		u, _ := url.Parse(config.DbURL)
		parts := strings.Split(u.Host, ":")
		return parts[0]
	}

	return config.DbHost
}

// GetDbPort - Gets the database port from the given configuration
func (config DatabaseConfig) GetDbPort() int {
	if config.DbURL != "" {
		u, _ := url.Parse(config.DbURL)
		parts := strings.Split(u.Host, ":")

		if len(parts) == 2 {
			port, _ := strconv.Atoi(parts[1])
			return port
		}

		return 5432
	}

	return config.DbPort
}
