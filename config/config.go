package config

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bmizerany/lpx"
)

type Config struct {
	// Only used for Heroku servers to pass log messages to the input handler
	HerokuLogStream chan HerokuLogStreamItem

	Servers []ServerConfig
}

type HerokuLogStreamItem struct {
	Header    lpx.Header
	Content   []byte
	Namespace string
}

type ServerIdentifier struct {
	APIKey      string
	APIBaseURL  string
	SystemID    string
	SystemType  string
	SystemScope string
}

// ServerConfig -
//   Contains the information how to connect to a Postgres instance,
//   with optional AWS credentials to get metrics
//   from AWS CloudWatch as well as RDS logfiles
type ServerConfig struct {
	APIKey     string `ini:"api_key"`
	APIBaseURL string `ini:"api_base_url"`

	ErrorCallback   string `ini:"error_callback"`
	SuccessCallback string `ini:"success_callback"`

	EnableReports   bool `ini:"enable_reports"`
	DisableLogs     bool `ini:"disable_logs"`
	DisableActivity bool `ini:"disable_activity"`

	DbURL                 string `ini:"db_url"`
	DbName                string `ini:"db_name"`
	DbUsername            string `ini:"db_username"`
	DbPassword            string `ini:"db_password"`
	DbHost                string `ini:"db_host"`
	DbPort                int    `ini:"db_port"`
	DbSslMode             string `ini:"db_sslmode"`
	DbSslRootCert         string `ini:"db_sslrootcert"`
	DbSslRootCertContents string `ini:"db_sslrootcert_contents"`

	// We have to do some tricks to support sslmode=prefer, namely we have to
	// first try an SSL connection (= require), and if that fails change the
	// sslmode to none
	DbSslModePreferFailed bool

	DbExtraNames []string // Additional databases that should be fetched (determined by additional databases in db_name)
	DbAllNames   bool     // All databases except template databases should be fetched (determined by * in the db_name list)

	AwsRegion          string `ini:"aws_region"`
	AwsDbInstanceID    string `ini:"aws_db_instance_id"`
	AwsAccessKeyID     string `ini:"aws_access_key_id"`
	AwsSecretAccessKey string `ini:"aws_secret_access_key"`

	// Support for custom AWS endpoints
	// See https://docs.aws.amazon.com/sdk-for-go/api/aws/endpoints/
	AwsEndpointSigningRegion     string `ini:"aws_endpoint_rds_signing_region"`
	AwsEndpointRdsURL            string `ini:"aws_endpoint_rds_url"`
	AwsEndpointEc2URL            string `ini:"aws_endpoint_ec2_url"`
	AwsEndpointCloudwatchURL     string `ini:"aws_endpoint_cloudwatch_url"`
	AwsEndpointCloudwatchLogsURL string `ini:"aws_endpoint_cloudwatch_logs_url"`

	AzureDbServerName string `ini:"azure_db_server_name"`

	GcpProjectID          string `ini:"gcp_project_id"`
	GcpCloudSQLInstanceID string `ini:"gcp_cloudsql_instance_id"`

	SectionName string
	Identifier  ServerIdentifier

	SystemID    string `ini:"api_system_id"`
	SystemType  string `ini:"api_system_type"`
	SystemScope string `ini:"api_system_scope"`

	// Configures the location where logfiles are - this can either be a directory,
	// or a file - needs to readable by the regular pganalyze user
	LogLocation string `ini:"db_log_location"`

	// Configures the collector to tail a local docker container using
	// "docker logs -t" - this is currently experimental and mostly intended for
	// development and debugging. The value needs to be the name of the container.
	LogDockerTail string `ini:"db_log_docker_tail"`

	// Specifies a table pattern to ignore - no statistics will be collected for
	// tables that match the name. This uses Golang's filepath.Match function for
	// comparison, so you can e.g. use "*" for wildcard matching.
	IgnoreTablePattern string `ini:"ignore_table_pattern"`

	// Specifies the frequency of query statistics collection in seconds
	//
	// Currently supported values: 600 (10 minutes), 60 (1 minute)
	//
	// Defaults to once per minute (60)
	QueryStatsInterval int `ini:"query_stats_interval"`

	// Maximum connections allowed to the database with the collector
	// application_name, in order to protect against accidental connection leaks
	// in the collector
	//
	// This defaults to 10 connections, but you may want to raise this when running
	// the collector multiple times against the same database server
	MaxCollectorConnections int `ini:"max_collector_connections"`

	// Configuration for PII filtering
	FilterLogSecret   string `ini:"filter_log_secret"`   // none/all/credential/parsing_error/statement_text/statement_parameter/table_data/ops/unidentified (comma separated)
	FilterQuerySample string `ini:"filter_query_sample"` // none/all

	// HttpClient - Client to be used for API connections
	HTTPClient *http.Client
}

// GetPqOpenString - Gets the database configuration as a string that can be passed to lib/pq for connecting
func (config ServerConfig) GetPqOpenString(dbNameOverride string) string {
	var dbUsername, dbPassword, dbName, dbHost, dbSslMode, dbSslRootCert string
	var dbPort int

	if config.DbURL != "" {
		u, _ := url.Parse(config.DbURL)

		if u.User != nil {
			dbUsername = u.User.Username()
			dbPassword, _ = u.User.Password()
		}

		if u.Path != "" {
			dbName = u.Path[1:len(u.Path)]
		}

		hostSplits := strings.SplitN(u.Host, ":", 2)
		dbHost = hostSplits[0]
		if len(hostSplits) > 1 {
			dbPort, _ = strconv.Atoi(hostSplits[1])
		}

		querySplits := strings.Split(u.RawQuery, "&")
		for _, querySplit := range querySplits {
			keyValue := strings.SplitN(querySplit, "=", 2)
			switch keyValue[0] {
			case "sslmode":
				dbSslMode = keyValue[1]
			case "sslrootcert":
				dbSslRootCert = keyValue[1]
			}
		}
	}

	dbinfo := []string{}

	if config.DbUsername != "" {
		dbUsername = config.DbUsername
	}
	if config.DbPassword != "" {
		dbPassword = config.DbPassword
	}
	if dbNameOverride != "" {
		dbName = dbNameOverride
	} else if config.DbName != "" {
		dbName = config.DbName
	}
	if config.DbHost != "" {
		dbHost = config.DbHost
	}
	if config.DbPort != 0 {
		dbPort = config.DbPort
	}
	if config.DbSslMode != "" {
		dbSslMode = config.DbSslMode
	}
	if config.DbSslRootCert != "" {
		dbSslRootCert = config.DbSslRootCert
	}

	// Defaults if nothing is set
	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbPort == 0 {
		dbPort = 5432
	}
	if dbSslMode == "" {
		dbSslMode = "prefer"
	}

	// Handle SSL mode prefer
	if dbSslMode == "prefer" {
		if config.DbSslModePreferFailed {
			dbSslMode = "disable"
		} else {
			dbSslMode = "require"
		}
	}

	// Handle SSL certificates shipped with the collector
	if dbSslRootCert == "rds-ca-2015-root" {
		dbSslRootCert = "/usr/share/pganalyze-collector/sslrootcert/rds-ca-2015-root.pem"
	}
	if dbSslRootCert == "rds-ca-2019-root" {
		dbSslRootCert = "/usr/share/pganalyze-collector/sslrootcert/rds-ca-2019-root.pem"
	}

	// Generate the actual string
	if dbUsername != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("user='%s'", strings.Replace(dbUsername, "'", "\\'", -1)))
	}
	if dbPassword != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("password='%s'", strings.Replace(dbPassword, "'", "\\'", -1)))
	}
	if dbName != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("dbname='%s'", strings.Replace(dbName, "'", "\\'", -1)))
	}
	if dbHost != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("host='%s'", strings.Replace(dbHost, "'", "\\'", -1)))
	}
	if dbPort != 0 {
		dbinfo = append(dbinfo, fmt.Sprintf("port=%d", dbPort))
	}
	if dbSslMode != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("sslmode=%s", dbSslMode))
	}
	if dbSslRootCert != "" {
		dbinfo = append(dbinfo, fmt.Sprintf("sslrootcert='%s'", strings.Replace(dbSslRootCert, "'", "\\'", -1)))
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

// GetDbUsername - Gets the database hostname from the given configuration
func (config ServerConfig) GetDbUsername() string {
	if config.DbURL != "" {
		u, _ := url.Parse(config.DbURL)
		if u != nil && u.User != nil {
			return u.User.Username()
		}
	}

	return config.DbUsername
}

// GetDbName - Gets the database name from the given configuration
func (config ServerConfig) GetDbName() string {
	if config.DbURL != "" {
		u, _ := url.Parse(config.DbURL)
		if len(u.Path) > 0 {
			return u.Path[1:len(u.Path)]
		}
	}

	return config.DbName
}
