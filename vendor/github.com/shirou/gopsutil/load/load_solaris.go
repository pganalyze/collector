// +build solaris

package load

import "github.com/shirou/gopsutil/internal/common"

func Avg() (*AvgStat, error) {
	return nil, common.ErrNotImplementedError
}
