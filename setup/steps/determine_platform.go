package steps

import (
	"errors"

	s "github.com/pganalyze/collector/setup/state"
	"github.com/shirou/gopsutil/host"
)

var DeterminePlatform = &s.Step{
	Description: "Determine platform",
	Check: func(state *s.SetupState) (bool, error) {
		hostInfo, err := host.Info()
		if err != nil {
			return false, err
		}
		state.OperatingSystem = hostInfo.OS
		state.Platform = hostInfo.Platform
		state.PlatformFamily = hostInfo.PlatformFamily
		state.PlatformVersion = hostInfo.PlatformVersion

		// TODO: relax this
		if state.Platform != "ubuntu" || state.PlatformVersion != "20.04" {
			return false, errors.New("not supported on platforms other than Ubuntu 20.04")
		}

		return true, nil
	},
}
