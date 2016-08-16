// +build solaris

package host

import "github.com/shirou/gopsutil/internal/common"

func Info() (*InfoStat, error) {
	return nil, common.ErrNotImplementedError
}
