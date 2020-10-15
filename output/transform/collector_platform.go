package transform

import (
	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformCollectorPlatform(s snapshot.FullSnapshot, state state.TransientState) snapshot.FullSnapshot {
	p := state.CollectorPlatform
	startTs, _ := ptypes.TimestampProto(p.StartedAt)
	s.CollectorStartedAt = startTs
	s.CollectorArchitecture = p.Architecture
	s.CollectorHostname = p.Hostname
	s.CollectorOperatingSystem = p.OperatingSystem
	s.CollectorPlatform = p.Platform
	s.CollectorPlatformFamily = p.PlatformFamily
	s.CollectorPlatformVersion = p.PlatformVersion
	s.CollectorVirtualizationSystem = p.VirtualizationSystem
	s.CollectorKernelVersion = p.KernelVersion
	return s
}
