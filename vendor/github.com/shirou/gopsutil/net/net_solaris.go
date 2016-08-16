// +build solaris

package net

import "github.com/shirou/gopsutil/internal/common"

func IOCounters(pernic bool) ([]IOCountersStat, error) {
	return []IOCountersStat{}, common.ErrNotImplementedError
}
