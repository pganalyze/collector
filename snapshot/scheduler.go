//go:generate msgp

package snapshot

// Scheduler - Information about the OS scheduler
type Scheduler struct {
	ContextSwitches NullableInt `msg:"context_switches,omitempty"`
	Interrupts      NullableInt `msg:"interrupts,omitempty"`

	Loadavg1min  NullableFloat `msg:"loadavg_1min,omitempty"`
	Loadavg5min  NullableFloat `msg:"loadavg_5min,omitempty"`
	Loadavg15min NullableFloat `msg:"loadavg_15min,omitempty"`

	ProcsBlocked NullableInt `msg:"procs_blocked,omitempty"`
	ProcsCreated NullableInt `msg:"procs_created,omitempty"`
	ProcsRunning NullableInt `msg:"procs_running,omitempty"`
}
