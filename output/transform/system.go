package transform

import (
	"sort"

	"github.com/golang/protobuf/ptypes"
	snapshot "github.com/pganalyze/collector/output/pganalyze_collector"
	"github.com/pganalyze/collector/state"
)

func SystemStateToCompactSystemSnapshot(systemState state.SystemState) snapshot.CompactSystemSnapshot {
	// TODO: We should probably have the caller handle this - if we wanted to
	// support compact system snapshots for self-hosted systems there needs to be
	// actual diff-ing going on
	diffState := state.DiffState{}
	diffState.SystemDiskStats = make(state.DiffedDiskStatsMap)
	for deviceName, stats := range systemState.DiskStats {
		if stats.DiffedOnInput && stats.DiffedValues != nil {
			diffState.SystemDiskStats[deviceName] = *stats.DiffedValues
		}
	}

	system := transformSystem(systemState, diffState)
	return snapshot.CompactSystemSnapshot{System: system}
}

func systemStateToFullSnapshot(s snapshot.FullSnapshot, newState state.PersistedState, diffState state.DiffState) snapshot.FullSnapshot {
	s.System = transformSystem(newState.System, diffState)
	return s
}

func transformSystem(systemState state.SystemState, diffState state.DiffState) *snapshot.System {
	system := &snapshot.System{}

	system.SystemInformation = &snapshot.SystemInformation{}

	if systemState.Info.Type == state.SelfHostedSystem {
		system.SystemInformation.Type = snapshot.SystemInformation_SELF_HOSTED_SYSTEM
		if systemState.Info.SelfHosted != nil {
			system.SystemInformation.Info = &snapshot.SystemInformation_SelfHosted{
				SelfHosted: &snapshot.SystemInformationSelfHosted{
					Hostname:                 systemState.Info.SelfHosted.Hostname,
					Architecture:             systemState.Info.SelfHosted.Architecture,
					OperatingSystem:          systemState.Info.SelfHosted.OperatingSystem,
					Platform:                 systemState.Info.SelfHosted.Platform,
					PlatformFamily:           systemState.Info.SelfHosted.PlatformFamily,
					PlatformVersion:          systemState.Info.SelfHosted.PlatformVersion,
					VirtualizationSystem:     systemState.Info.SelfHosted.VirtualizationSystem,
					KernelVersion:            systemState.Info.SelfHosted.KernelVersion,
					DatabaseSystemIdentifier: systemState.Info.SelfHosted.DatabaseSystemIdentifier,
				},
			}
		}
	} else if systemState.Info.Type == state.AmazonRdsSystem {
		system.SystemInformation.Type = snapshot.SystemInformation_AMAZON_RDS_SYSTEM
		if systemState.Info.AmazonRds != nil {
			latestRestorableTime, _ := ptypes.TimestampProto(systemState.Info.AmazonRds.LatestRestorableTime)
			createdAt, _ := ptypes.TimestampProto(systemState.Info.AmazonRds.CreatedAt)

			system.SystemInformation.Info = &snapshot.SystemInformation_AmazonRds{
				AmazonRds: &snapshot.SystemInformationAmazonRDS{
					Region:                     systemState.Info.AmazonRds.Region,
					InstanceClass:              systemState.Info.AmazonRds.InstanceClass,
					InstanceId:                 systemState.Info.AmazonRds.InstanceID,
					Status:                     systemState.Info.AmazonRds.Status,
					AvailabilityZone:           systemState.Info.AmazonRds.AvailabilityZone,
					PubliclyAccessible:         systemState.Info.AmazonRds.PubliclyAccessible,
					MultiAz:                    systemState.Info.AmazonRds.MultiAz,
					SecondaryAvailabilityZone:  systemState.Info.AmazonRds.SecondaryAvailabilityZone,
					CaCertificate:              systemState.Info.AmazonRds.CaCertificate,
					AutoMinorVersionUpgrade:    systemState.Info.AmazonRds.AutoMinorVersionUpgrade,
					PreferredMaintenanceWindow: systemState.Info.AmazonRds.PreferredMaintenanceWindow,
					PreferredBackupWindow:      systemState.Info.AmazonRds.PreferredBackupWindow,
					LatestRestorableTime:       latestRestorableTime,
					BackupRetentionPeriodDays:  systemState.Info.AmazonRds.BackupRetentionPeriodDays,
					MasterUsername:             systemState.Info.AmazonRds.MasterUsername,
					InitialDbName:              systemState.Info.AmazonRds.InitialDbName,
					CreatedAt:                  createdAt,
					EnhancedMonitoring:         systemState.Info.AmazonRds.EnhancedMonitoring,
					ParameterApplyStatus:       systemState.Info.AmazonRds.ParameterApplyStatus,
					ParameterPgssEnabled:       systemState.Info.AmazonRds.ParameterPgssEnabled,
				},
			}
		}
	} else if systemState.Info.Type == state.HerokuSystem {
		system.SystemInformation.Type = snapshot.SystemInformation_HEROKU_SYSTEM
	}

	system.SystemId = systemState.Info.SystemID
	system.SystemScope = systemState.Info.SystemScope
	system.XlogUsedBytes = systemState.XlogUsedBytes

	system.SchedulerStatistic = &snapshot.SchedulerStatistic{
		LoadAverage_1Min:  systemState.Scheduler.Loadavg1min,
		LoadAverage_5Min:  systemState.Scheduler.Loadavg5min,
		LoadAverage_15Min: systemState.Scheduler.Loadavg15min,
	}

	system.MemoryStatistic = &snapshot.MemoryStatistic{
		ApplicationBytes:   systemState.Memory.ApplicationBytes,
		TotalBytes:         systemState.Memory.TotalBytes,
		CachedBytes:        systemState.Memory.CachedBytes,
		BuffersBytes:       systemState.Memory.BuffersBytes,
		FreeBytes:          systemState.Memory.FreeBytes,
		WritebackBytes:     systemState.Memory.WritebackBytes,
		DirtyBytes:         systemState.Memory.DirtyBytes,
		SlabBytes:          systemState.Memory.SlabBytes,
		MappedBytes:        systemState.Memory.MappedBytes,
		PageTablesBytes:    systemState.Memory.PageTablesBytes,
		ActiveBytes:        systemState.Memory.ActiveBytes,
		InactiveBytes:      systemState.Memory.InactiveBytes,
		AvailableBytes:     systemState.Memory.AvailableBytes,
		SwapUsedBytes:      systemState.Memory.SwapUsedBytes,
		SwapTotalBytes:     systemState.Memory.SwapTotalBytes,
		HugePagesSizeBytes: systemState.Memory.HugePagesSizeBytes,
		HugePagesFree:      systemState.Memory.HugePagesFree,
		HugePagesTotal:     systemState.Memory.HugePagesTotal,
		HugePagesReserved:  systemState.Memory.HugePagesReserved,
		HugePagesSurplus:   systemState.Memory.HugePagesSurplus,
	}

	system.CpuInformation = &snapshot.CPUInformation{
		Model:             systemState.CPUInfo.Model,
		CacheSizeBytes:    systemState.CPUInfo.CacheSizeBytes,
		SpeedMhz:          systemState.CPUInfo.SpeedMhz,
		SocketCount:       systemState.CPUInfo.SocketCount,
		PhysicalCoreCount: systemState.CPUInfo.PhysicalCoreCount,
		LogicalCoreCount:  systemState.CPUInfo.LogicalCoreCount,
	}

	for cpuID, cpuStats := range diffState.SystemCPUStats {
		ref := snapshot.CPUReference{
			CoreId: cpuID,
		}
		idx := int32(len(system.CpuReferences))
		system.CpuReferences = append(system.CpuReferences, &ref)
		system.CpuStatistics = append(system.CpuStatistics, &snapshot.CPUStatistic{
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

	interfaceNames := []string{}
	for k := range diffState.SystemNetworkStats {
		interfaceNames = append(interfaceNames, k)
	}
	sort.Strings(interfaceNames)

	for _, interfaceName := range interfaceNames {
		interfaceStats := diffState.SystemNetworkStats[interfaceName]
		ref := snapshot.NetworkReference{
			InterfaceName: interfaceName,
		}
		idx := int32(len(system.NetworkReferences))
		system.NetworkReferences = append(system.NetworkReferences, &ref)
		system.NetworkStatistics = append(system.NetworkStatistics, &snapshot.NetworkStatistic{
			NetworkIdx:                       idx,
			TransmitThroughputBytesPerSecond: interfaceStats.TransmitThroughputBytesPerSecond,
			ReceiveThroughputBytesPerSecond:  interfaceStats.ReceiveThroughputBytesPerSecond,
		})
	}

	deviceNames := []string{}
	for k := range systemState.Disks {
		deviceNames = append(deviceNames, k)
	}
	sort.Strings(deviceNames)

	diskNameToIdx := make(map[string]int32)

	for _, deviceName := range deviceNames {
		disk := systemState.Disks[deviceName]
		ref := snapshot.DiskReference{
			DeviceName: deviceName,
		}
		idx := int32(len(system.DiskReferences))
		system.DiskReferences = append(system.DiskReferences, &ref)
		diskNameToIdx[deviceName] = idx

		system.DiskInformations = append(system.DiskInformations, &snapshot.DiskInformation{
			DiskIdx:         idx,
			DiskType:        disk.DiskType,
			Scheduler:       disk.Scheduler,
			ProvisionedIops: disk.ProvisionedIOPS,
			Encrypted:       disk.Encrypted,
		})

		diskStats, exists := diffState.SystemDiskStats[deviceName]
		if exists {
			system.DiskStatistics = append(system.DiskStatistics, &snapshot.DiskStatistic{
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

	mountpoints := []string{}
	for k := range systemState.DiskPartitions {
		mountpoints = append(mountpoints, k)
	}
	sort.Strings(mountpoints)

	for _, mountpoint := range mountpoints {
		diskPartition := systemState.DiskPartitions[mountpoint]
		ref := snapshot.DiskPartitionReference{
			Mountpoint: mountpoint,
		}
		idx := int32(len(system.DiskPartitionReferences))
		system.DiskPartitionReferences = append(system.DiskPartitionReferences, &ref)

		diskIdx := diskNameToIdx[diskPartition.DiskName]

		system.DiskPartitionInformations = append(system.DiskPartitionInformations, &snapshot.DiskPartitionInformation{
			DiskPartitionIdx: idx,
			DiskIdx:          diskIdx,
			FilesystemType:   diskPartition.FilesystemType,
			FilesystemOpts:   diskPartition.FilesystemOpts,
			PartitionName:    diskPartition.PartitionName,
		})

		system.DiskPartitionStatistics = append(system.DiskPartitionStatistics, &snapshot.DiskPartitionStatistic{
			DiskPartitionIdx: idx,
			UsedBytes:        diskPartition.UsedBytes,
			TotalBytes:       diskPartition.TotalBytes,
		})

		if mountpoint == systemState.DataDirectoryPartition {
			system.DataDirectoryDiskPartitionIdx = idx
		}
		if mountpoint == systemState.XlogPartition {
			system.XlogDiskPartitionIdx = idx
		}
	}

	return system
}
