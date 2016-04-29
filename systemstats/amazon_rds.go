//go:generate msgp

package systemstats

import (
	"encoding/json"
	"fmt"

	"gopkg.in/guregu/null.v2"

	//"github.com/aws/aws-sdk-go/aws/request"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/snapshot"
	"github.com/pganalyze/collector/util"
)

// GetFromAmazonRds - Gets system information about an Amazon RDS instance
func getFromAmazonRds(config config.DatabaseConfig) (system *snapshot.System) {
	sess := util.GetAwsSession(config)

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
		SystemType: snapshot.AmazonRdsSystem,
	}

	systemInfo := &snapshot.AmazonRdsInfo{
		Region:                    snapshot.NullableString(null.StringFrom(config.AwsRegion)),
		InstanceClass:             snapshot.NullableString(null.StringFromPtr(instance.DBInstanceClass)),
		InstanceID:                snapshot.NullableString(null.StringFromPtr(instance.DBInstanceIdentifier)),
		Status:                    snapshot.NullableString(null.StringFromPtr(instance.DBInstanceStatus)),
		AvailabilityZone:          snapshot.NullableString(null.StringFromPtr(instance.AvailabilityZone)),
		PubliclyAccessible:        snapshot.NullableBool(null.BoolFromPtr(instance.PubliclyAccessible)),
		MultiAZ:                   snapshot.NullableBool(null.BoolFromPtr(instance.MultiAZ)),
		SecondaryAvailabilityZone: snapshot.NullableString(null.StringFromPtr(instance.SecondaryAvailabilityZone)),
		CACertificate:             snapshot.NullableString(null.StringFromPtr(instance.CACertificateIdentifier)),

		AutoMinorVersionUpgrade:    snapshot.NullableBool(null.BoolFromPtr(instance.AutoMinorVersionUpgrade)),
		PreferredMaintenanceWindow: snapshot.NullableString(null.StringFromPtr(instance.PreferredMaintenanceWindow)),

		LatestRestorableTime:  snapshot.NullableUnixTimestamp(util.TimestampFromPtr(instance.LatestRestorableTime)),
		PreferredBackupWindow: snapshot.NullableString(null.StringFromPtr(instance.PreferredBackupWindow)),
		BackupRetentionPeriod: snapshot.NullableInt(null.IntFromPtr(instance.BackupRetentionPeriod)),

		MasterUsername: snapshot.NullableString(null.StringFromPtr(instance.MasterUsername)),
		InitialDbName:  snapshot.NullableString(null.StringFromPtr(instance.DBName)),
		CreatedAt:      snapshot.NullableUnixTimestamp(util.TimestampFromPtr(instance.InstanceCreateTime)),

		StorageProvisionedIops: snapshot.NullableInt(null.IntFromPtr(instance.Iops)),
		StorageEncrypted:       snapshot.NullableBool(null.BoolFromPtr(instance.StorageEncrypted)),
		StorageType:            snapshot.NullableString(null.StringFromPtr(instance.StorageType)),
	}

	group := instance.DBParameterGroups[0]

	pgssParam, _ := util.GetRdsParameter(group, "shared_preload_libraries", rdsSvc)

	systemInfo.ParameterPgssEnabled = pgssParam != nil && *pgssParam.ParameterValue == "pg_stat_statements"
	systemInfo.ParameterApplyStatus = *group.ParameterApplyStatus

	system.SystemInfo = systemInfo

	// Not fetched right now:
	// - DatabaseConnections
	// - TransactionLogsDiskUsage

	dbInstanceID := *instance.DBInstanceIdentifier

	system.CPU.Utilization = util.GetRdsFloatMetric(dbInstanceID, "CPUUtilization", "Percent", sess)

	system.Network = &snapshot.Network{
		ReceiveThroughput:  snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "NetworkReceiveThroughput", "Bytes/Second", sess))),
		TransmitThroughput: snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "NetworkTransmitThroughput", "Bytes/Second", sess))),
	}

	storage := snapshot.Storage{
		BytesAvailable: snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "FreeStorageSpace", "Bytes", sess))),
		Perfdata: snapshot.StoragePerfdata{
			Version:         1,
			ReadIops:        snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "ReadIOPS", "Count/Second", sess))),
			WriteIops:       snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "WriteIOPS", "Count/Second", sess))),
			ReadThroughput:  snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "ReadThroughput", "Bytes/Second", sess))),
			WriteThroughput: snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "WriteThroughput", "Bytes/Second", sess))),
			IopsInProgress:  snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "DiskQueueDepth", "Count", sess))),
			ReadLatency:     snapshot.NullableFloat(null.FloatFromPtr(util.GetRdsFloatMetric(dbInstanceID, "ReadLatency", "Seconds", sess))),
			WriteLatency:    snapshot.NullableFloat(null.FloatFromPtr(util.GetRdsFloatMetric(dbInstanceID, "WriteLatency", "Seconds", sess))),
		},
	}

	var bytesTotal int64
	if instance.AllocatedStorage != nil {
		bytesTotal = *instance.AllocatedStorage * 1024 * 1024 * 1024
		storage.BytesTotal = snapshot.NullableInt(null.IntFrom(bytesTotal))
	}

	system.Storage = append(system.Storage, storage)

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
				var osSnapshot snapshot.RdsOsSnapshot
				err = json.Unmarshal([]byte(*str), &osSnapshot)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					return
				}

				// Technically these are not msec but percentages, so we multiply by the number of milliseconds in a minute (our standard measurement)
				system.CPU.BusyTimesGuestMsec = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.CPUUtilization.Guest * 60000)))
				system.CPU.BusyTimesGuestNiceMsec = snapshot.NullableInt(null.IntFrom(0))
				system.CPU.BusyTimesIdleMsec = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.CPUUtilization.Idle * 60000)))
				system.CPU.BusyTimesSoftirqMsec = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.CPUUtilization.Irq * 60000)))
				system.CPU.BusyTimesIrqMsec = snapshot.NullableInt(null.IntFrom(0))
				system.CPU.BusyTimesIowaitMsec = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.CPUUtilization.Wait * 60000)))
				system.CPU.BusyTimesSystemMsec = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.CPUUtilization.System * 60000)))
				system.CPU.BusyTimesUserMsec = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.CPUUtilization.User * 60000)))
				system.CPU.BusyTimesStealMsec = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.CPUUtilization.Steal * 60000)))
				system.CPU.BusyTimesNiceMsec = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.CPUUtilization.Nice * 60000)))

				system.CPU.HardwareSockets = snapshot.NullableInt(null.IntFrom(1))
				system.CPU.HardwareCoresPerSocket = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.NumVCPUs)))

				system.Scheduler.Loadavg1min = snapshot.NullableFloat(null.FloatFrom(float64(osSnapshot.LoadAverageMinute.One)))
				system.Scheduler.Loadavg5min = snapshot.NullableFloat(null.FloatFrom(float64(osSnapshot.LoadAverageMinute.Five)))
				system.Scheduler.Loadavg15min = snapshot.NullableFloat(null.FloatFrom(float64(osSnapshot.LoadAverageMinute.Fifteen)))

				system.Scheduler.ProcsRunning = snapshot.NullableInt(null.IntFrom(osSnapshot.Tasks.Running))
				system.Scheduler.ProcsBlocked = snapshot.NullableInt(null.IntFrom(osSnapshot.Tasks.Blocked))

				system.Memory.ApplicationsBytes = snapshot.NullableInt(null.IntFrom((osSnapshot.Memory.Total - osSnapshot.Memory.Free - osSnapshot.Memory.Buffers - osSnapshot.Memory.Cached) * 1024))
				system.Memory.BuffersBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Memory.Buffers * 1024))
				system.Memory.DirtyBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Memory.Dirty * 1024))
				system.Memory.FreeBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Memory.Free * 1024))
				system.Memory.PagecacheBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Memory.Cached * 1024))
				system.Memory.SwapFreeBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Swap.Free * 1024))
				system.Memory.SwapTotalBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Swap.Total * 1024))
				system.Memory.TotalBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Memory.Total * 1024))
				system.Memory.WritebackBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Memory.Writeback * 1024))
				system.Memory.ActiveBytes = snapshot.NullableInt(null.IntFrom(osSnapshot.Memory.Active * 1024))

				system.Storage[0].Perfdata.AvgReqSize = snapshot.NullableInt(null.IntFrom(int64(osSnapshot.DiskIO[0].AvgReqSz * 1024)))

				systemInfo.OsSnapshot = &osSnapshot
			}
		}
	} else {
		system.Memory.FreeBytes = snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "FreeableMemory", "Bytes", sess)))
		system.Memory.SwapTotalBytes = snapshot.NullableInt(null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "SwapUsage", "Bytes", sess)))
		system.Memory.SwapFreeBytes = snapshot.NullableInt(null.IntFrom(0))
	}

	return
}
