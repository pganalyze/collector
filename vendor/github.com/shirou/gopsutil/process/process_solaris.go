// +build solaris

package process

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/internal/common"
)

func Pids() ([]int32, error) {
	return []int32{}, common.ErrNotImplementedError
}

func (p *Process) Times() (*cpu.TimesStat, error) {
	return nil, common.ErrNotImplementedError
}

func (p *Process) MemoryInfo() (*MemoryInfoStat, error) {
	return nil, common.ErrNotImplementedError
}

func NewProcess(pid int32) (*Process, error) {
	return nil, common.ErrNotImplementedError
}
