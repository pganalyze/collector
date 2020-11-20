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

		if state.Platform == "ubuntu" {
			if state.PlatformVersion < "14.04" {
				return false, errors.New("Ubuntu versions older than 14.04 are not supported")
			}
		} else if state.Platform == "debian" {
			if state.PlatformVersion < "10.0" {
				return false, errors.New("Debian versions older than 10 are not supported")
			}
		} else {
			return false, errors.New("this distribution is not currently supported; please contact support")
		}

		return true, nil
	},
}
