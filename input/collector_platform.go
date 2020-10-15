package input

import (
	"runtime"

	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/shirou/gopsutil/host"
)

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
