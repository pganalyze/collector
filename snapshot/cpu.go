//go:generate msgp

package snapshot

// CPU - Information about CPU activity
type CPU struct {
	Utilization *float64 `msg:"utilization"`

	BusyTimesGuestMsec     NullableInt `msg:"busy_times_guest_msec,omitempty"`
	BusyTimesGuestNiceMsec NullableInt `msg:"busy_times_guest_nice_msec,omitempty"`
	BusyTimesIdleMsec      NullableInt `msg:"busy_times_idle_msec,omitempty"`
	BusyTimesIowaitMsec    NullableInt `msg:"busy_times_iowait_msec,omitempty"`
	BusyTimesIrqMsec       NullableInt `msg:"busy_times_irq_msec,omitempty"`
	BusyTimesNiceMsec      NullableInt `msg:"busy_times_nice_msec,omitempty"`
	BusyTimesSoftirqMsec   NullableInt `msg:"busy_times_softirq_msec,omitempty"`
	BusyTimesStealMsec     NullableInt `msg:"busy_times_steal_msec,omitempty"`
	BusyTimesSystemMsec    NullableInt `msg:"busy_times_system_msec,omitempty"`
	BusyTimesUserMsec      NullableInt `msg:"busy_times_user_msec,omitempty"`

	HardwareCacheSize      NullableString `msg:"hardware_cache_size,omitempty"`
	HardwareModel          NullableString `msg:"hardware_model,omitempty"`
	HardwareSockets        NullableInt    `msg:"hardware_sockets,omitempty"`
	HardwareCoresPerSocket NullableInt    `msg:"hardware_cores_per_socket,omitempty"`
	HardwareSpeedMhz       NullableFloat  `msg:"hardware_speed_mhz,omitempty"`
}
