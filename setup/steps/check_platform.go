package steps

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/pganalyze/collector/setup/state"
	"github.com/shirou/gopsutil/host"
)

var CheckPlatform = &state.Step{
	ID:          "check_platform",
	Description: "Check whether this platform is supported by pganalyze guided setup",
	Check: func(s *state.SetupState) (bool, error) {
		hostInfo, err := host.Info()
		if err != nil {
			return false, err
		}
		s.OperatingSystem = hostInfo.OS
		s.Platform = hostInfo.Platform
		s.PlatformFamily = hostInfo.PlatformFamily
		s.PlatformVersion = hostInfo.PlatformVersion

		platVerNum, err := strconv.ParseFloat(s.PlatformVersion, 32)
		if err != nil {
			return false, fmt.Errorf("could not parse current platform version: %s / version %s", s.Platform, s.PlatformVersion)
		}

		if s.Platform == "ubuntu" {
			if platVerNum < 14.04 {
				return false, errors.New("Ubuntu versions older than 14.04 are not supported")
			}
		} else if s.Platform == "debian" {
			if platVerNum < 10.0 {
				return false, errors.New("Debian versions older than 10 are not supported")
			}
		} else {
			return false, fmt.Errorf("the current platform (%s) is not currently supported; please contact support", s.Platform)
		}

		return true, nil
	},
}
