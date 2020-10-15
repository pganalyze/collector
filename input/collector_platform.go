package input

import (
	"runtime"

	"github.com/pganalyze/collector/state"
	"github.com/shirou/gopsutil/host"
)

func getCollectorPlatform(globalCollectionOpts state.CollectionOpts) state.CollectorPlatform {
	hostInfo, err := host.Info()
	if err != nil {
		// TODO: log this? return error?
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
