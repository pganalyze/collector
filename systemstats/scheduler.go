package systemstats

import "gopkg.in/guregu/null.v2"

// Scheduler - Information about the OS scheduler
type Scheduler struct {
	ContextSwitches null.Int `json:"context_switches,omitempty"`
	Interrupts      null.Int `json:"interrupts,omitempty"`

	Loadavg1min  null.Float `json:"loadavg_1min,omitempty"`
	Loadavg5min  null.Float `json:"loadavg_5min,omitempty"`
	Loadavg15min null.Float `json:"loadavg_15min,omitempty"`

	ProcsBlocked null.Int `json:"procs_blocked,omitempty"`
	ProcsCreated null.Int `json:"procs_created,omitempty"`
	ProcsRunning null.Int `json:"procs_running,omitempty"`
}
