package input

import (
	"os"
	"runtime"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/process"
)

func getMemoryRssBytes() uint64 {
	pid := os.Getpid()

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return 0
	}

	mem, err := p.MemoryInfo()
	if err != nil {
		return 0
	}

	return mem.RSS
}

func getCollectorStats() state.CollectorStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return state.CollectorStats{
		GoVersion:                runtime.Version(),
		ActiveGoroutines:         int32(runtime.NumGoroutine()),
		CgoCalls:                 runtime.NumCgoCall(),
		MemoryHeapAllocatedBytes: memStats.HeapAlloc,
		MemoryHeapObjects:        memStats.HeapObjects,
		MemorySystemBytes:        memStats.Sys,
		MemoryRssBytes:           getMemoryRssBytes(),
	}
}

func getCollectorPlatform(globalCollectionOpts state.CollectionOpts, logger *util.Logger) state.CollectorPlatform {
	hostInfo, err := host.Info()
	if err != nil {
		if globalCollectionOpts.TestRun {
			logger.PrintVerbose("Could not get collector host information: %s", err)
		}
		return state.CollectorPlatform{}
	}

	var virtSystem string
	if hostInfo.VirtualizationRole == "guest" {
		virtSystem = hostInfo.VirtualizationSystem
	}
	return state.CollectorPlatform{
		StartedAt:            globalCollectionOpts.StartedAt,
		Architecture:         runtime.GOARCH,
		Hostname:             hostInfo.Hostname,
		OperatingSystem:      hostInfo.OS,
		Platform:             hostInfo.Platform,
		PlatformFamily:       hostInfo.PlatformFamily,
		PlatformVersion:      hostInfo.PlatformVersion,
		KernelVersion:        hostInfo.KernelVersion,
		VirtualizationSystem: virtSystem,
	}
}

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
