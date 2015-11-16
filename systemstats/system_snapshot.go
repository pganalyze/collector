package systemstats

import "github.com/lfittl/pganalyze-collector-next/config"

// SystemSnapshot - All kinds of system-related information and metrics
type SystemSnapshot struct {
	SystemType SystemType  `json:"system_type"`
	SystemInfo interface{} `json:"system_info,omitempty"`
	Storage    []Storage   `json:"storage"`
	CPU        CPU         `json:"cpu"`
	Memory     Memory      `json:"memory"`
	Network    *Network    `json:"network,omitempty"`
}

// SystemType - Enum that describes which kind of system we're monitoring
type SystemType int

// Treat this list as append-only and never change the order.
const (
	PhysicalSystem SystemType = iota
	VirtualSystem
	AmazonRdsSystem
	HerokuSystem
)

// GetSystemSnapshot - Retrieves a system snapshot for this system and returns it
func GetSystemSnapshot(config config.Config) (system *SystemSnapshot) {
	// TODO: We need a smarter selection mechanism here, and also consider AWS instances by hostname
	if config.AwsDbInstanceId != "" {
		system = getFromAmazonRds(config)
	}

	return
}
