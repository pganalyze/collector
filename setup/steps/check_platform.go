package steps

import (
	"errors"
	"fmt"
	"strconv"

	s "github.com/pganalyze/collector/setup/state"
	"github.com/shirou/gopsutil/host"
)

var CheckPlatform = &s.Step{
	ID:          "check_platform",
	Description: "Check whether this platform is supported by pganalyze guided setup",
	Check: func(state *s.SetupState) (bool, error) {
		hostInfo, err := host.Info()
		if err != nil {
			return false, err
		}
		state.OperatingSystem = hostInfo.OS
		state.Platform = hostInfo.Platform
		state.PlatformFamily = hostInfo.PlatformFamily
		state.PlatformVersion = hostInfo.PlatformVersion

		platVerNum, err := strconv.ParseFloat(state.PlatformVersion, 32)
		if err != nil {
			return false, fmt.Errorf("could not parse current platform version: %s / version %s", state.Platform, state.PlatformVersion)
		}

		if state.Platform == "ubuntu" {
			if platVerNum < 14.04 {
				return false, errors.New("Ubuntu versions older than 14.04 are not supported")
			}
		} else if state.Platform == "debian" {
			if platVerNum < 10.0 {
				return false, errors.New("Debian versions older than 10 are not supported")
			}
		} else {
			return false, fmt.Errorf("the current platform (%s) is not currently supported; please contact support", state.Platform)
		}

		return true, nil
	},
}
