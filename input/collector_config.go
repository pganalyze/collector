package input

import (
	"os"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
)

func getCollectorConfig(c config.ServerConfig) state.CollectorConfig {
	return state.CollectorConfig{
		SectionName:             c.SectionName,
		DisableLogs:             c.DisableLogs,
		DisableActivity:         c.DisableActivity,
		EnableLogExplain:        c.EnableLogExplain,
		DbName:                  c.DbName,
		DbUsername:              c.DbUsername,
		DbHost:                  c.DbHost,
		DbPort:                  int32(c.DbPort),
		DbSslmode:               c.DbSslMode,
		DbHasSslrootcert:        c.DbSslRootCert != "" || c.DbSslRootCertContents != "",
		DbHasSslcert:            c.DbSslCert != "" || c.DbSslCertContents != "",
		DbHasSslkey:             c.DbSslKey != "" || c.DbSslKeyContents != "",
		DbExtraNames:            c.DbExtraNames,
		DbAllNames:              c.DbAllNames,
		AwsRegion:               c.AwsRegion,
		AwsDbInstanceId:         c.AwsDbInstanceID,
		AwsHasAccessKeyId:       c.AwsAccessKeyID != "",
		AzureDbServerName:       c.AzureDbServerName,
		AzureEventhubNamespace:  c.AzureEventhubNamespace,
		AzureEventhubName:       c.AzureEventhubName,
		AzureAdTenantId:         c.AzureADTenantID,
		AzureAdClientId:         c.AzureADClientID,
		AzureHasAdCertificate:   c.AzureADCertificatePath != "",
		GcpCloudsqlInstanceId:   c.GcpCloudSQLInstanceID,
		GcpPubsubSubscription:   c.GcpPubsubSubscription,
		GcpHasCredentialsFile:   c.GcpCredentialsFile != "",
		GcpProjectId:            c.GcpProjectID,
		ApiSystemId:             c.SystemID,
		ApiSystemType:           c.SystemType,
		ApiSystemScope:          c.SystemScope,
		DbLogLocation:           c.LogLocation,
		DbLogDockerTail:         c.LogDockerTail,
		IgnoreTablePattern:      c.IgnoreTablePattern,
		IgnoreSchemaRegexp:      c.IgnoreSchemaRegexp,
		QueryStatsInterval:      int32(c.QueryStatsInterval),
		MaxCollectorConnections: int32(c.MaxCollectorConnections),
		FilterLogSecret:         c.FilterLogSecret,
		FilterQuerySample:       c.FilterQuerySample,
		HasProxy:                c.HTTPProxy != "" || c.HTTPSProxy != "",
		ConfigFromEnv:           os.Getenv("PGA_API_KEY") != "",
	}
}
