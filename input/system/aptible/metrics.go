package aptible

import (
	"context"
	"time"

	"github.com/pganalyze/collector/grant"
	"github.com/pganalyze/collector/output"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

const MB_TO_BYTE = 1024 * 1024

type AptibleMetric struct {
	Running       bool   `json:"running"`
	MilliCpuUsage int    `json:"milli_cpu_usage"` // the Container's average CPU usage (in milli CPUs) over the reporting period.
	MilliCpuLimit int    `json:"milli_cpu_limit"` // the maximum CPU accessible to the container. If CPU Isolation is disabled, this metric will return 0.
	MemoryTotalMB int    `json:"memory_total_mb"` // the Container's total memory usage.
	MemoryRssMB   int    `json:"memory_rss_mb"`   // the Container's RSS memory usage. This memory is typically not reclaimable. If this exceeds the memory_limit_mb, the container will be restarted.
	MemoryLimitMB int    `json:"memory_limit_mb"` // the Container's Memory Limit.
	DiskReadKBPS  int    `json:"disk_read_kbps"`  // the Container's average disk read bandwidth over the reporting period.
	DiskWriteKBPS int    `json:"disk_write_kbps"` // the Container's average disk write bandwidth over the reporting period.
	DiskReadIOPS  int    `json:"disk_read_iops"`  // the Container's average disk read IOPS over the reporting period.
	DiskWriteIOPS int    `json:"disk_write_iops"` // the Container's average disk write IOPS over the reporting period.
	DiskUsageMB   int    `json:"disk_usage_mb"`   // the Database's Disk usage (Database metrics only).
	DiskLimitMB   int    `json:"disk_limit_mb"`   // the Database's Disk size (Database metrics only).
	PidsCurrent   int    `json:"pids_current"`    // the current number of tasks in the Container (see Other Limits).
	PidsLimit     int    `json:"pids_limit"`      // the maximum number of tasks for the Container (see Other Limits).
	Environment   string `json:"environment"`     // Environment handle
	App           string `json:"app"`             // App handle (App metrics only)
	Database      string `json:"database"`        // Database handle (Database metrics only)
	Service       string `json:"service"`         // Service name
	HostName      string `json:"host_name"`       // Container Hostname (Short Container ID)
	Container     string `json:"container"`       // full Container ID
}

func HandleMetricMessage(ctx context.Context, sample *AptibleMetric, globalCollectionOpts state.CollectionOpts, logger *util.Logger, servers []*state.Server) {
	if sample.Database != "healthie-staging-14" {
		return
	}

	for _, server := range servers {
		if server.Config.SectionName == "healthie-staging-14" {
			server.CollectionStatusMutex.Lock()
			if server.CollectionStatus.CollectionDisabled {
				server.CollectionStatusMutex.Unlock()
				return
			}
			server.CollectionStatusMutex.Unlock()

			prefixedLogger := logger.WithPrefix(server.Config.SectionName)

			grant, err := grant.GetDefaultGrant(server, globalCollectionOpts, prefixedLogger)
			if err != nil {
				prefixedLogger.PrintError("Could not get default grant for system snapshot: %s", err)
				return
			}

			system := state.SystemState{}
			system.Info.Type = state.SelfHostedSystem
			system.Info.SystemID = server.Config.SystemID
			system.Info.SystemScope = server.Config.SystemScope
			system.Scheduler = state.Scheduler{
				Loadavg1min:  float64(sample.MilliCpuUsage / sample.MilliCpuLimit),
				Loadavg5min:  float64(sample.MilliCpuUsage / sample.MilliCpuLimit),
				Loadavg15min: float64(sample.MilliCpuUsage / sample.MilliCpuLimit),
			}

			system.Memory = state.Memory{
				ApplicationBytes: uint64(sample.MemoryRssMB * MB_TO_BYTE),
				TotalBytes:       uint64(sample.MemoryTotalMB * MB_TO_BYTE),
				FreeBytes:        uint64((sample.MemoryLimitMB - sample.MemoryRssMB) * MB_TO_BYTE),
				CachedBytes:      uint64((sample.MemoryTotalMB - sample.MemoryRssMB) * MB_TO_BYTE),
			}

			system.Disks = make(state.DiskMap)
			system.Disks["default"] = state.Disk{}

			system.DiskPartitions = make(state.DiskPartitionMap)
			system.DiskPartitions["/"] = state.DiskPartition{
				DiskName:   "default",
				UsedBytes:  uint64(sample.DiskUsageMB * MB_TO_BYTE),
				TotalBytes: uint64(sample.DiskLimitMB * MB_TO_BYTE),
			}

			system.DiskStats = make(state.DiskStatsMap)
			system.DiskStats["default"] = state.DiskStats{
				DiffedOnInput: true,
				DiffedValues: &state.DiffedDiskStats{
					ReadOperationsPerSecond:  float64(sample.DiskReadIOPS),
					WriteOperationsPerSecond: float64(sample.DiskWriteIOPS),
				},
			}

			err = output.SubmitCompactSystemSnapshot(ctx, server, grant, globalCollectionOpts, prefixedLogger, system, time.Now())
			if err != nil {
				prefixedLogger.PrintError("Failed to upload/send compact metric snapshot: %s", err)
				return
			} else {
				prefixedLogger.PrintVerbose("Submitting metric message")
			}
		}
	}
}
