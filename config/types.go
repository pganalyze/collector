package config

import (
	"net/url"
	"strconv"
	"strings"
)

type DatabaseConfig struct {
	APIKey     string `ini:"api_key"`
	APIURL     string `ini:"api_url"`
	DbURL      string `ini:"db_url"`
	DbName     string `ini:"db_name"`
	DbUsername string `ini:"db_username"`
	DbPassword string `ini:"db_password"`
	DbHost     string `ini:"db_host"`
	DbPort     int    `ini:"db_port"`

	AwsRegion          string `ini:"aws_region"`
	AwsDbInstanceId    string `ini:"aws_db_instance_id"`
	AwsAccessKeyId     string `ini:"aws_access_key_id"`
	AwsSecretAccessKey string `ini:"aws_secret_access_key"`

	SectionName string
}

func (config DatabaseConfig) GetDbHost() string {
	if config.DbURL != "" {
		u, _ := url.Parse(config.DbURL)
		parts := strings.Split(u.Host, ":")
		return parts[0]
	}

	return config.DbHost
}

func (config DatabaseConfig) GetDbPort() int {
	if config.DbURL != "" {
		u, _ := url.Parse(config.DbURL)
		parts := strings.Split(u.Host, ":")

		if len(parts) == 2 {
			port, _ := strconv.Atoi(parts[1])
			return port
		} else {
			return 5432
		}
	}

	return config.DbPort
}
