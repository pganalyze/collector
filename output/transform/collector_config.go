package transform

import (
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformCollectorConfig(s snapshot.FullSnapshot, state state.TransientState) snapshot.FullSnapshot {
	c := state.CollectorConfig
	s.Config = &snapshot.CollectorConfig{
		SectionName:                c.SectionName,
		DisableLogs:                c.DisableLogs,
		DisableActivity:            c.DisableActivity,
		EnableLogExplain:           c.EnableLogExplain,
		DbName:                     c.DbName,
		DbUsername:                 c.DbUsername,
		DbHost:                     c.DbHost,
		DbPort:                     c.DbPort,
		DbSslmode:                  c.DbSslmode,
		DbHasSslrootcert:           c.DbHasSslrootcert,
		DbHasSslcert:               c.DbHasSslcert,
		DbHasSslkey:                c.DbHasSslkey,
		DbExtraNames:               c.DbExtraNames,
		DbAllNames:                 c.DbAllNames,
		DbUrl:                      c.DbURLRedacted,
		AwsRegion:                  c.AwsRegion,
		AwsDbInstanceId:            c.AwsDbInstanceId,
		AwsHasAccountId:            c.AwsHasAccountId,
		AwsHasAccessKeyId:          c.AwsHasAccessKeyId,
		AwsHasAssumeRole:           c.AwsHasAssumeRole,
		AwsHasWebIdentityTokenFile: c.AwsHasWebIdentityTokenFile,
		AwsHasRoleArn:              c.AwsHasRoleArn,
		AzureDbServerName:          c.AzureDbServerName,
		AzureEventhubNamespace:     c.AzureEventhubNamespace,
		AzureEventhubName:          c.AzureEventhubName,
		AzureAdTenantId:            c.AzureAdTenantId,
		AzureAdClientId:            c.AzureAdClientId,
		AzureHasAdCertificate:      c.AzureHasAdCertificate,
		GcpCloudsqlInstanceId:      c.GcpCloudsqlInstanceId,
		GcpPubsubSubscription:      c.GcpPubsubSubscription,
		GcpHasCredentialsFile:      c.GcpHasCredentialsFile,
		GcpProjectId:               c.GcpProjectId,
		CrunchyBridgeClusterId:     c.CrunchyBridgeClusterId,
		AivenProjectId:             c.AivenProjectId,
		AivenServiceId:             c.AivenServiceId,
		ApiSystemId:                c.ApiSystemId,
		ApiSystemType:              c.ApiSystemType,
		ApiSystemScope:             c.ApiSystemScope,
		ApiSystemScopeFallback:     c.ApiSystemScopeFallback,
		DbLogLocation:              c.DbLogLocation,
		DbLogDockerTail:            c.DbLogDockerTail,
		IgnoreTablePattern:         c.IgnoreTablePattern,
		IgnoreSchemaRegexp:         c.IgnoreSchemaRegexp,
		QueryStatsInterval:         c.QueryStatsInterval,
		MaxCollectorConnections:    c.MaxCollectorConnections,
		SkipIfReplica:              c.SkipIfReplica,
		FilterLogSecret:            c.FilterLogSecret,
		FilterQuerySample:          c.FilterQuerySample,
		FilterQueryText:            c.FilterQueryText,
		HasProxy:                   c.HasProxy,
		ConfigFromEnv:              c.ConfigFromEnv,
	}
	return s
}
