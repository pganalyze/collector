package transform

import (
	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func transformSystem(s snapshot.FullSnapshot, newState state.State, diffState state.DiffState) snapshot.FullSnapshot {
	s.System = &snapshot.System{}

	s.System.SystemInformation = &snapshot.SystemInformation{}

	if newState.System.Info.Type == state.SelfHostedSystem {
		s.System.SystemInformation.Type = snapshot.SystemInformation_SELF_HOSTED_SYSTEM
		if newState.System.Info.SelfHosted != nil {
			s.System.SystemInformation.Info = &snapshot.SystemInformation_SelfHosted{
				SelfHosted: &snapshot.SystemInformationSelfHosted{
					Hostname:             newState.System.Info.SelfHosted.Hostname,
					Architecture:         newState.System.Info.SelfHosted.Architecture,
					OperatingSystem:      newState.System.Info.SelfHosted.OperatingSystem,
					Platform:             newState.System.Info.SelfHosted.Platform,
					PlatformFamily:       newState.System.Info.SelfHosted.PlatformFamily,
					PlatformVersion:      newState.System.Info.SelfHosted.PlatformVersion,
					VirtualizationSystem: newState.System.Info.SelfHosted.VirtualizationSystem,
					KernelVersion:        newState.System.Info.SelfHosted.KernelVersion,
				},
			}
		}
	} else if newState.System.Info.Type == state.AmazonRdsSystem {
		s.System.SystemInformation.Type = snapshot.SystemInformation_AMAZON_RDS_SYSTEM
		if newState.System.Info.AmazonRds != nil {
			latestRestorableTime, _ := ptypes.TimestampProto(newState.System.Info.AmazonRds.LatestRestorableTime)
			createdAt, _ := ptypes.TimestampProto(newState.System.Info.AmazonRds.CreatedAt)

			s.System.SystemInformation.Info = &snapshot.SystemInformation_AmazonRds{
				AmazonRds: &snapshot.SystemInformationAmazonRDS{
					Region:                     newState.System.Info.AmazonRds.Region,
					InstanceClass:              newState.System.Info.AmazonRds.InstanceClass,
					InstanceId:                 newState.System.Info.AmazonRds.InstanceID,
					Status:                     newState.System.Info.AmazonRds.Status,
					AvailabilityZone:           newState.System.Info.AmazonRds.AvailabilityZone,
					PubliclyAccessible:         newState.System.Info.AmazonRds.PubliclyAccessible,
					MultiAz:                    newState.System.Info.AmazonRds.MultiAz,
					SecondaryAvailabilityZone:  newState.System.Info.AmazonRds.SecondaryAvailabilityZone,
					CaCertificate:              newState.System.Info.AmazonRds.CaCertificate,
					AutoMinorVersionUpgrade:    newState.System.Info.AmazonRds.AutoMinorVersionUpgrade,
					PreferredMaintenanceWindow: newState.System.Info.AmazonRds.PreferredMaintenanceWindow,
					PreferredBackupWindow:      newState.System.Info.AmazonRds.PreferredBackupWindow,
					LatestRestorableTime:       latestRestorableTime,
					BackupRetentionPeriodDays:  newState.System.Info.AmazonRds.BackupRetentionPeriodDays,
					MasterUsername:             newState.System.Info.AmazonRds.MasterUsername,
					InitialDbName:              newState.System.Info.AmazonRds.InitialDbName,
					CreatedAt:                  createdAt,
					EnhancedMonitoring:         newState.System.Info.AmazonRds.EnhancedMonitoring,
					ParameterApplyStatus:       newState.System.Info.AmazonRds.ParameterApplyStatus,
					ParameterPgssEnabled:       newState.System.Info.AmazonRds.ParameterPgssEnabled,
				},
			}
		}
	} else if newState.System.Info.Type == state.HerokuSystem {
		s.System.SystemInformation.Type = snapshot.SystemInformation_HEROKU_SYSTEM
		// TODO: Add Info
	}

	s.System.XlogUsedBytes = newState.System.XlogUsedBytes

	s.System.SchedulerStatistic = &snapshot.SchedulerStatistic{
		LoadAverage_1Min:  newState.System.Scheduler.Loadavg1min,
		LoadAverage_5Min:  newState.System.Scheduler.Loadavg5min,
		LoadAverage_15Min: newState.System.Scheduler.Loadavg15min,
	}

	s.System.MemoryStatistic = &snapshot.MemoryStatistic{
		TotalBytes:         newState.System.Memory.TotalBytes,
		CachedBytes:        newState.System.Memory.CachedBytes,
		BuffersBytes:       newState.System.Memory.BuffersBytes,
		FreeBytes:          newState.System.Memory.FreeBytes,
		WritebackBytes:     newState.System.Memory.WritebackBytes,
		DirtyBytes:         newState.System.Memory.DirtyBytes,
		SlabBytes:          newState.System.Memory.SlabBytes,
		MappedBytes:        newState.System.Memory.MappedBytes,
		PageTablesBytes:    newState.System.Memory.PageTablesBytes,
		ActiveBytes:        newState.System.Memory.ActiveBytes,
		InactiveBytes:      newState.System.Memory.InactiveBytes,
		AvailableBytes:     newState.System.Memory.AvailableBytes,
		SwapUsedBytes:      newState.System.Memory.SwapUsedBytes,
		SwapTotalBytes:     newState.System.Memory.SwapTotalBytes,
		HugePagesSizeBytes: newState.System.Memory.HugePagesSizeBytes,
		HugePagesFree:      newState.System.Memory.HugePagesFree,
		HugePagesTotal:     newState.System.Memory.HugePagesTotal,
		HugePagesReserved:  newState.System.Memory.HugePagesReserved,
		HugePagesSurplus:   newState.System.Memory.HugePagesSurplus,
	}

	s.System.CpuInformation = &snapshot.CPUInformation{
		Model:             newState.System.CPUInfo.Model,
		CacheSizeBytes:    newState.System.CPUInfo.CacheSizeBytes,
		SpeedMhz:          newState.System.CPUInfo.SpeedMhz,
		SocketCount:       newState.System.CPUInfo.SocketCount,
		PhysicalCoreCount: newState.System.CPUInfo.PhysicalCoreCount,
		LogicalCoreCount:  newState.System.CPUInfo.LogicalCoreCount,
	}

	for cpuID, cpuStats := range diffState.SystemCPUStats {
		ref := snapshot.CPUReference{
			CoreId: cpuID,
		}
		idx := int32(len(s.System.CpuReferences))
		s.System.CpuReferences = append(s.System.CpuReferences, &ref)
		s.System.CpuStatistics = append(s.System.CpuStatistics, &snapshot.CPUStatistic{
			CpuIdx:           idx,
			UserPercent:      cpuStats.UserPercent,
			SystemPercent:    cpuStats.SystemPercent,
			IdlePercent:      cpuStats.IdlePercent,
			NicePercent:      cpuStats.NicePercent,
			IowaitPercent:    cpuStats.IowaitPercent,
			IrqPercent:       cpuStats.IrqPercent,
			SoftIrqPercent:   cpuStats.SoftIrqPercent,
			StealPercent:     cpuStats.StealPercent,
			GuestPercent:     cpuStats.GuestPercent,
			GuestNicePercent: cpuStats.GuestNicePercent,
		})
	}

	for interfaceName, interfaceStats := range diffState.SystemNetworkStats {
		ref := snapshot.NetworkReference{
			InterfaceName: interfaceName,
		}
		idx := int32(len(s.System.NetworkReferences))
		s.System.NetworkReferences = append(s.System.NetworkReferences, &ref)
		s.System.NetworkStatistics = append(s.System.NetworkStatistics, &snapshot.NetworkStatistic{
			NetworkIdx:                       idx,
			TransmitThroughputBytesPerSecond: interfaceStats.TransmitThroughputBytesPerSecond,
			ReceiveThroughputBytesPerSecond:  interfaceStats.ReceiveThroughputBytesPerSecond,
		})
	}

	diskNameToIdx := make(map[string]int32)

	for deviceName, disk := range newState.System.Disks {
		ref := snapshot.DiskReference{
			DeviceName: deviceName,
		}
		idx := int32(len(s.System.DiskReferences))
		s.System.DiskReferences = append(s.System.DiskReferences, &ref)
		diskNameToIdx[deviceName] = idx

		s.System.DiskInformations = append(s.System.DiskInformations, &snapshot.DiskInformation{
			DiskIdx:         idx,
			DiskType:        disk.DiskType,
			Scheduler:       disk.Scheduler,
			ProvisionedIops: disk.ProvisionedIOPS,
			Encrypted:       disk.Encrypted,
		})

		diskStats, exists := diffState.SystemDiskStats[deviceName]
		if exists {
			s.System.DiskStatistics = append(s.System.DiskStatistics, &snapshot.DiskStatistic{
				DiskIdx:                 idx,
				ReadOperationsPerSecond: diskStats.ReadOperationsPerSecond,
				ReadsMergedPerSecond:    diskStats.ReadsMergedPerSecond,
				BytesReadPerSecond:      diskStats.BytesReadPerSecond,
				AvgReadLatency:          diskStats.AvgReadLatency,

				WriteOperationsPerSecond: diskStats.WriteOperationsPerSecond,
				WritesMergedPerSecond:    diskStats.WritesMergedPerSecond,
				BytesWrittenPerSecond:    diskStats.BytesWrittenPerSecond,
				AvgWriteLatency:          diskStats.AvgWriteLatency,
				AvgQueueSize:             diskStats.AvgQueueSize,
				UtilizationPercent:       diskStats.UtilizationPercent,
			})
		}
	}

	for mountpoint, diskPartition := range newState.System.DiskPartitions {
		ref := snapshot.DiskPartitionReference{
			Mountpoint: mountpoint,
		}
		idx := int32(len(s.System.DiskPartitionReferences))
		s.System.DiskPartitionReferences = append(s.System.DiskPartitionReferences, &ref)

		diskIdx := diskNameToIdx[diskPartition.DiskName]

		s.System.DiskPartitionInformations = append(s.System.DiskPartitionInformations, &snapshot.DiskPartitionInformation{
			DiskPartitionIdx: idx,
			DiskIdx:          diskIdx,
			FilesystemType:   diskPartition.FilesystemType,
			FilesystemOpts:   diskPartition.FilesystemOpts,
			PartitionName:    diskPartition.PartitionName,
		})

		s.System.DiskPartitionStatistics = append(s.System.DiskPartitionStatistics, &snapshot.DiskPartitionStatistic{
			DiskPartitionIdx: idx,
			UsedBytes:        diskPartition.UsedBytes,
			TotalBytes:       diskPartition.TotalBytes,
		})
	}

	return s
}
