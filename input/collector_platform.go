package input

import (
	"runtime"
	"time"

	"github.com/pganalyze/collector/state"
	"github.com/shirou/gopsutil/host"
)

// TODO: is there a better place to initialize this?
var collectorStartTime = time.Now()

func getCollectorPlatform() state.CollectorPlatform {
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
		StartedAt:            collectorStartTime,
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
