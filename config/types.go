package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ServerConfig -
//   Contains the information how to connect to a Postgres instance,
//   with optional AWS credentials to get metrics
//   from AWS CloudWatch as well as RDS logfiles
type ServerConfig struct {
	APIKey     string `ini:"api_key"`
	APIBaseURL string `ini:"api_base_url"`
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
func (config ServerConfig) GetPqOpenString() string {
	if config.DbURL != "" {
		return config.DbURL
	}

	dbinfo := []string{}

	if config.DbUsername != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("user=%s", config.DbUsername))
	}
	if config.DbPassword != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("password=%s", config.DbPassword))
	}
	if config.DbName != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("dbname=%s", config.DbName))
	}
	if config.DbHost != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("host=%s", config.DbHost))
	}
	if config.DbPort != 0 {
		dbinfo = append(dbinfo, fmt.Sprintf("port=%d", config.DbPort))
	}
	if config.DbSslMode != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("sslmode=%s", config.DbSslMode))
	}
	dbinfo = append(dbinfo, "connect_timeout=10")

	return strings.Join(dbinfo, " ")
}

// GetDbHost - Gets the database hostname from the given configuration
func (config ServerConfig) GetDbHost() string {
	if config.DbURL != "" {
		u, _ := url.Parse(config.DbURL)
		parts := strings.Split(u.Host, ":")
		return parts[0]
	}

	return config.DbHost
}

// GetDbPort - Gets the database port from the given configuration
func (config ServerConfig) GetDbPort() int {
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
