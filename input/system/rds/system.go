package rds

import (
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
)

// GetSystemState - Gets system information about an Amazon RDS instance
func GetSystemState(config config.ServerConfig, logger *util.Logger) (system state.SystemState) {
	/*sess := util.GetAwsSession(config)

	rdsSvc := rds.New(sess)

	instance, err := util.FindRdsInstance(config, sess)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if instance == nil {
		fmt.Println("Could not find RDS instance in AWS, skipping system data")
		return
	}

	system = &snapshot.System{
		SystemType: snapshot.SystemType_PHYSICAL_SYSTEM,
	}

	systemInfo := &snapshot.AmazonRdsInfo{
		Region:                    config.AwsRegion,
		InstanceClass:             util.StringPtrToString(instance.DBInstanceClass),
		InstanceId:                util.StringPtrToString(instance.DBInstanceIdentifier),
		Status:                    util.StringPtrToString(instance.DBInstanceStatus),
		AvailabilityZone:          util.StringPtrToString(instance.AvailabilityZone),
		PubliclyAccessible:        util.BoolPtrToBool(instance.PubliclyAccessible),
		MultiAz:                   util.BoolPtrToBool(instance.MultiAZ),
		SecondaryAvailabilityZone: util.StringPtrToString(instance.SecondaryAvailabilityZone),
		CaCertificate:             util.StringPtrToString(instance.CACertificateIdentifier),

		AutoMinorVersionUpgrade:    util.BoolPtrToBool(instance.AutoMinorVersionUpgrade),
		PreferredMaintenanceWindow: util.StringPtrToString(instance.PreferredMaintenanceWindow),

		LatestRestorableTime:  util.TimePtrToUnixTimestamp(instance.LatestRestorableTime),
		PreferredBackupWindow: util.StringPtrToString(instance.PreferredBackupWindow),
		BackupRetentionPeriod: util.IntPtrToString(instance.BackupRetentionPeriod),

		MasterUsername: util.StringPtrToString(instance.MasterUsername),
		InitialDbName:  util.StringPtrToString(instance.DBName),
		CreatedAt:      util.TimePtrToUnixTimestamp(instance.InstanceCreateTime),

		StorageProvisionedIops: util.IntPtrToString(instance.Iops),
		StorageEncrypted:       util.BoolPtrToBool(instance.StorageEncrypted),
		StorageType:            util.StringPtrToString(instance.StorageType),
	}

	group := instance.DBParameterGroups[0]

	pgssParam, _ := util.GetRdsParameter(group, "shared_preload_libraries", rdsSvc)

	systemInfo.ParameterPgssEnabled = pgssParam != nil && *pgssParam.ParameterValue == "pg_stat_statements"
	systemInfo.ParameterApplyStatus = *group.ParameterApplyStatus

	system.SystemInfo = &snapshot.System_AmazonRdsInfo{AmazonRdsInfo: systemInfo}

	// Not fetched right now:
	// - DatabaseConnections
	// - TransactionLogsDiskUsage

	dbInstanceID := *instance.DBInstanceIdentifier

	cloudWatchReader := util.NewRdsCloudWatchReader(sess, logger, dbInstanceID)

	system.Cpu = &snapshot.CPU{
		Utilization: cloudWatchReader.GetRdsFloatMetric("CPUUtilization", "Percent"),
	}

	system.Network = &snapshot.Network{
		ReceiveThroughput:  cloudWatchReader.GetRdsIntMetric("NetworkReceiveThroughput", "Bytes/Second"),
		TransmitThroughput: cloudWatchReader.GetRdsIntMetric("NetworkTransmitThroughput", "Bytes/Second"),
	}

	storage := snapshot.Storage{
		BytesAvailable: cloudWatchReader.GetRdsIntMetric("FreeStorageSpace", "Bytes"),
		Perfdata: &snapshot.StoragePerfdata{
			Version:      1,
			RdIos:        cloudWatchReader.GetRdsIntMetric("ReadIOPS", "Count/Second"),
			WrIos:        cloudWatchReader.GetRdsIntMetric("WriteIOPS", "Count/Second"),
			RdThroughput: cloudWatchReader.GetRdsIntMetric("ReadThroughput", "Bytes/Second"),
			WrThroughput: cloudWatchReader.GetRdsIntMetric("WriteThroughput", "Bytes/Second"),
			IosInProg:    cloudWatchReader.GetRdsIntMetric("DiskQueueDepth", "Count"),
			RdLatency:    cloudWatchReader.GetRdsFloatMetric("ReadLatency", "Seconds"),
			WrLatency:    cloudWatchReader.GetRdsFloatMetric("WriteLatency", "Seconds"),
		},
	}

	var bytesTotal int64
	if instance.AllocatedStorage != nil {
		bytesTotal = *instance.AllocatedStorage * 1024 * 1024 * 1024
		storage.BytesTotal = bytesTotal
	}

	system.Storage = append(system.Storage, &storage)

	if instance.EnhancedMonitoringResourceArn != nil {
		svc := cloudwatchlogs.New(sess)

		params := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String("RDSOSMetrics"),
			LogStreamName: instance.DbiResourceId,
			Limit:         aws.Int64(1),
		}

		resp, err := svc.GetLogEvents(params)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		event := resp.Events[0]
		if event != nil {
			str := event.Message
			if str != nil {
				var osSnapshot RdsOsSnapshot
				err = json.Unmarshal([]byte(*str), &osSnapshot)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					return
				}

				// Technically these are not msec but percentages, so we multiply by the number of milliseconds in a minute (our standard measurement)
				system.Cpu = &snapshot.CPU{}
				system.Cpu.BusyTimesGuestMsec = int64(osSnapshot.CPUUtilization.Guest * 60000)
				system.Cpu.BusyTimesGuestNiceMsec = 0
				system.Cpu.BusyTimesIdleMsec = int64(osSnapshot.CPUUtilization.Idle * 60000)
				system.Cpu.BusyTimesSoftirqMsec = int64(osSnapshot.CPUUtilization.Irq * 60000)
				system.Cpu.BusyTimesIrqMsec = 0
				system.Cpu.BusyTimesIowaitMsec = int64(osSnapshot.CPUUtilization.Wait * 60000)
				system.Cpu.BusyTimesSystemMsec = int64(osSnapshot.CPUUtilization.System * 60000)
				system.Cpu.BusyTimesUserMsec = int64(osSnapshot.CPUUtilization.User * 60000)
				system.Cpu.BusyTimesStealMsec = int64(osSnapshot.CPUUtilization.Steal * 60000)
				system.Cpu.BusyTimesNiceMsec = int64(osSnapshot.CPUUtilization.Nice * 60000)
				system.Cpu.HardwareSockets = 1
				system.Cpu.HardwareCoresPerSocket = int64(osSnapshot.NumVCPUs)

				system.Scheduler = &snapshot.Scheduler{}
				system.Scheduler.Loadavg_1Min = float64(osSnapshot.LoadAverageMinute.One)
				system.Scheduler.Loadavg_5Min = float64(osSnapshot.LoadAverageMinute.Five)
				system.Scheduler.Loadavg_15Min = float64(osSnapshot.LoadAverageMinute.Fifteen)
				system.Scheduler.ProcsRunning = osSnapshot.Tasks.Running
				system.Scheduler.ProcsBlocked = osSnapshot.Tasks.Blocked

				system.Memory = &snapshot.Memory{}
				system.Memory.ApplicationsBytes = (osSnapshot.Memory.Total - osSnapshot.Memory.Free - osSnapshot.Memory.Buffers - osSnapshot.Memory.Cached) * 1024
				system.Memory.BuffersBytes = osSnapshot.Memory.Buffers * 1024
				system.Memory.DirtyBytes = osSnapshot.Memory.Dirty * 1024
				system.Memory.FreeBytes = osSnapshot.Memory.Free * 1024
				system.Memory.PagecacheBytes = osSnapshot.Memory.Cached * 1024
				system.Memory.SwapFreeBytes = osSnapshot.Swap.Free * 1024
				system.Memory.SwapTotalBytes = osSnapshot.Swap.Total * 1024
				system.Memory.TotalBytes = osSnapshot.Memory.Total * 1024
				system.Memory.WritebackBytes = osSnapshot.Memory.Writeback * 1024
				system.Memory.ActiveBytes = osSnapshot.Memory.Active * 1024

				system.Storage[0].Perfdata.AvgReqSize = int64(osSnapshot.DiskIO[0].AvgReqSz * 1024)
			}
		}
	} else {
		system.Memory = &snapshot.Memory{
			FreeBytes:      cloudWatchReader.GetRdsIntMetric("FreeableMemory", "Bytes"),
			SwapTotalBytes: cloudWatchReader.GetRdsIntMetric("SwapUsage", "Bytes"),
			SwapFreeBytes:  0,
		}
	}*/

	return
}
