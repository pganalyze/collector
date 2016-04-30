//go:generate msgp

package snapshot

// System - All kinds of system-related information and metrics
type System struct {
	SystemType SystemType  `msg:"system_type"`
	SystemInfo interface{} `msg:"system_info,omitempty"`
	Storage    []Storage   `msg:"storage"`
	CPU        CPU         `msg:"cpu"`
	Memory     Memory      `msg:"memory"`
	Network    *Network    `msg:"network,omitempty"`
	Scheduler  Scheduler   `msg:"scheduler,omitempty"`
}

// SystemType - Enum that describes which kind of system we're monitoring
type SystemType int

// Treat this list as append-only and never change the order
const (
	PhysicalSystem SystemType = iota
	VirtualSystem
	AmazonRdsSystem
	HerokuSystem
)
