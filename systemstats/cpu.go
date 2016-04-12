package systemstats

import "gopkg.in/guregu/null.v2"

// CPU - Information about CPU activity
type CPU struct {
	Utilization *float64 `json:"utilization"`

	BusyTimesGuestMsec     null.Int `json:"busy_times_guest_msec,omitempty"`
	BusyTimesGuestNiceMsec null.Int `json:"busy_times_guest_nice_msec,omitempty"`
	BusyTimesIdleMsec      null.Int `json:"busy_times_idle_msec,omitempty"`
	BusyTimesIowaitMsec    null.Int `json:"busy_times_iowait_msec,omitempty"`
	BusyTimesIrqMsec       null.Int `json:"busy_times_irq_msec,omitempty"`
	BusyTimesNiceMsec      null.Int `json:"busy_times_nice_msec,omitempty"`
	BusyTimesSoftirqMsec   null.Int `json:"busy_times_softirq_msec,omitempty"`
	BusyTimesStealMsec     null.Int `json:"busy_times_steal_msec,omitempty"`
	BusyTimesSystemMsec    null.Int `json:"busy_times_system_msec,omitempty"`
	BusyTimesUserMsec      null.Int `json:"busy_times_user_msec,omitempty"`

	HardwareCacheSize      *string    `json:"hardware_cache_size,omitempty"`
	HardwareModel          *string    `json:"hardware_model,omitempty"`
	HardwareSockets        null.Int   `json:"hardware_sockets,omitempty"`
	HardwareCoresPerSocket null.Int   `json:"hardware_cores_per_socket,omitempty"`
	HardwareSpeedMhz       null.Float `json:"hardware_speed_mhz,omitempty"`
}
