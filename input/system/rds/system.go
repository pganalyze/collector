package rds

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"
)

// Aurora storage is automatically extended up until 64TB, so we should always
// report that limit as the total disk space (to avoid bogus disk space warnings)
const AuroraMaxStorage = 64 * 1024 * 1024 * 1024 * 1024

// GetSystemState - Gets system information about an Amazon RDS instance
func GetSystemState(server *state.Server, logger *util.Logger) (system state.SystemState) {
	config := server.Config
	system.Info.Type = state.AmazonRdsSystem

	sess, err := awsutil.GetAwsSession(config)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting session: %v", err)
		logger.PrintError("Rds/System: Encountered error getting session: %v\n", err)
		return
	}

	rdsSvc := rds.New(sess)

	instance, err := awsutil.FindRdsInstance(config, sess)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error finding instance: %v", err)
		logger.PrintError("Rds/System: Encountered error when looking for instance: %v\n", err)
		return
	}

	system.Info.ClusterID = util.StringPtrToString(instance.DBClusterIdentifier)

	isAurora := util.StringPtrToString(instance.Engine) == "aurora-postgresql"

	system.Info.AmazonRds = &state.SystemInfoAmazonRds{
		Region:                     config.AwsRegion,
		InstanceClass:              util.StringPtrToString(instance.DBInstanceClass),
		InstanceID:                 util.StringPtrToString(instance.DBInstanceIdentifier),
		Status:                     util.StringPtrToString(instance.DBInstanceStatus),
		AvailabilityZone:           util.StringPtrToString(instance.AvailabilityZone),
		PubliclyAccessible:         util.BoolPtrToBool(instance.PubliclyAccessible),
		MultiAz:                    util.BoolPtrToBool(instance.MultiAZ),
		SecondaryAvailabilityZone:  util.StringPtrToString(instance.SecondaryAvailabilityZone),
		CaCertificate:              util.StringPtrToString(instance.CACertificateIdentifier),
		AutoMinorVersionUpgrade:    util.BoolPtrToBool(instance.AutoMinorVersionUpgrade),
		PreferredMaintenanceWindow: util.StringPtrToString(instance.PreferredMaintenanceWindow),
		PreferredBackupWindow:      util.StringPtrToString(instance.PreferredBackupWindow),
		LatestRestorableTime:       util.TimePtrToTime(instance.LatestRestorableTime),
		BackupRetentionPeriodDays:  int32(util.IntPtrToInt(instance.BackupRetentionPeriod)),
		MasterUsername:             util.StringPtrToString(instance.MasterUsername),
		InitialDbName:              util.StringPtrToString(instance.DBName),
		CreatedAt:                  util.TimePtrToTime(instance.InstanceCreateTime),
		PerformanceInsights:        util.BoolPtrToBool(instance.PerformanceInsightsEnabled),
		IAMAuthentication:          util.BoolPtrToBool(instance.IAMDatabaseAuthenticationEnabled),
		DeletionProtection:         util.BoolPtrToBool(instance.DeletionProtection),
		IsAuroraPostgres:           isAurora,
	}

	tags := make(map[string]string)
	for _, tag := range instance.TagList {
		tags[*tag.Key] = util.StringPtrToString(tag.Value)
	}

	system.Info.ResourceTags = tags

	for _, exportName := range instance.EnabledCloudwatchLogsExports {
		if util.StringPtrToString(exportName) == "postgresql" {
			system.Info.AmazonRds.PostgresLogExport = true
		}
	}

	group := instance.DBParameterGroups[0]

	pgssParam, err := awsutil.GetRdsParameter(group, "shared_preload_libraries", rdsSvc)
	if err != nil {
		server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting RDS parameter: %v", err)
		logger.PrintVerbose("Could not get RDS parameter: %s", err)
	}
	if pgssParam != nil && pgssParam.ParameterValue != nil {
		system.Info.AmazonRds.ParameterPgssEnabled = strings.Contains(*pgssParam.ParameterValue, "pg_stat_statements")
		system.Info.AmazonRds.ParameterAutoExplainEnabled = strings.Contains(*pgssParam.ParameterValue, "auto_explain")
	} else {
		system.Info.AmazonRds.ParameterPgssEnabled = false
		system.Info.AmazonRds.ParameterAutoExplainEnabled = false
	}
	system.Info.AmazonRds.ParameterApplyStatus = *group.ParameterApplyStatus

	dbInstanceID := *instance.DBInstanceIdentifier
	cloudWatchReader := awsutil.NewRdsCloudWatchReader(sess, logger, dbInstanceID)

	system.Disks = make(state.DiskMap)
	system.Disks["default"] = state.Disk{
		DiskType:        util.StringPtrToString(instance.StorageType),
		ProvisionedIOPS: uint32(util.IntPtrToInt(instance.Iops)),
		Encrypted:       util.BoolPtrToBool(instance.StorageEncrypted),
	}

	system.DiskStats = make(state.DiskStatsMap)
	system.DiskStats["default"] = state.DiskStats{
		DiffedOnInput: true,
		DiffedValues: &state.DiffedDiskStats{
			ReadOperationsPerSecond:  float64(cloudWatchReader.GetRdsIntMetric("ReadIOPS", "Count/Second")),
			WriteOperationsPerSecond: float64(cloudWatchReader.GetRdsIntMetric("WriteIOPS", "Count/Second")),
			BytesReadPerSecond:       float64(cloudWatchReader.GetRdsIntMetric("ReadThroughput", "Bytes/Second")),
			BytesWrittenPerSecond:    float64(cloudWatchReader.GetRdsIntMetric("WriteThroughput", "Bytes/Second")),
			AvgQueueSize:             int32(cloudWatchReader.GetRdsIntMetric("DiskQueueDepth", "Count")),
			AvgReadLatency:           cloudWatchReader.GetRdsFloatMetric("ReadLatency", "Seconds") * 1000,
			AvgWriteLatency:          cloudWatchReader.GetRdsFloatMetric("WriteLatency", "Seconds") * 1000,
		},
	}

	system.XlogUsedBytes = uint64(cloudWatchReader.GetRdsIntMetric("TransactionLogsDiskUsage", "Bytes"))

	if instance.EnhancedMonitoringResourceArn != nil {
		system.Info.AmazonRds.EnhancedMonitoring = true

		svc := cloudwatchlogs.New(sess)

		params := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String("RDSOSMetrics"),
			LogStreamName: instance.DbiResourceId,
			Limit:         aws.Int64(1),
		}

		resp, err := svc.GetLogEvents(params)
		if err != nil {
			server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error getting system state: %v", err)
			fmt.Printf("Error: %v\n", err)
			return
		}

		if len(resp.Events) > 0 {
			event := resp.Events[0]
			str := event.Message
			if str != nil {
				var osSnapshot RdsOsSnapshot
				err = json.Unmarshal([]byte(*str), &osSnapshot)
				if err != nil {
					server.SelfTest.MarkCollectionAspectError(state.CollectionAspectSystemStats, "error decoding system state: %v", err)
					fmt.Printf("Error: %v\n", err)
					return
				}

				system.CPUStats = make(state.CPUStatisticMap)
				system.CPUStats["all"] = state.CPUStatistic{
					DiffedOnInput: true,
					DiffedValues: &state.DiffedSystemCPUStats{
						GuestPercent:  float64(osSnapshot.CPUUtilization.Guest),
						IdlePercent:   float64(osSnapshot.CPUUtilization.Idle),
						IrqPercent:    float64(osSnapshot.CPUUtilization.Irq),
						IowaitPercent: float64(osSnapshot.CPUUtilization.Wait),
						SystemPercent: float64(osSnapshot.CPUUtilization.System),
						UserPercent:   float64(osSnapshot.CPUUtilization.User),
						StealPercent:  float64(osSnapshot.CPUUtilization.Steal),
						NicePercent:   float64(osSnapshot.CPUUtilization.Nice),
					},
				}

				system.CPUInfo.SocketCount = 1
				system.CPUInfo.LogicalCoreCount = int32(osSnapshot.NumVCPUs)
				system.CPUInfo.PhysicalCoreCount = int32(osSnapshot.NumVCPUs)

				system.Scheduler.Loadavg1min = float64(osSnapshot.LoadAverageMinute.One)
				system.Scheduler.Loadavg5min = float64(osSnapshot.LoadAverageMinute.Five)
				system.Scheduler.Loadavg15min = float64(osSnapshot.LoadAverageMinute.Fifteen)

				system.Memory.ActiveBytes = uint64(osSnapshot.Memory.Active * 1024)
				system.Memory.BuffersBytes = uint64(osSnapshot.Memory.Buffers * 1024)
				system.Memory.CachedBytes = uint64(osSnapshot.Memory.Cached * 1024)
				system.Memory.DirtyBytes = uint64(osSnapshot.Memory.Dirty * 1024)
				system.Memory.FreeBytes = uint64(osSnapshot.Memory.Free * 1024)
				system.Memory.HugePagesFree = uint64(osSnapshot.Memory.HugePagesFree)
				system.Memory.HugePagesReserved = uint64(osSnapshot.Memory.HugePagesRsvd)
				system.Memory.HugePagesSizeBytes = uint64(osSnapshot.Memory.HugePagesSize * 1024)
				system.Memory.HugePagesSurplus = uint64(osSnapshot.Memory.HugePagesSurp)
				system.Memory.HugePagesTotal = uint64(osSnapshot.Memory.HugePagesTotal)
				system.Memory.InactiveBytes = uint64(osSnapshot.Memory.Inactive * 1024)
				system.Memory.MappedBytes = uint64(osSnapshot.Memory.Mapped * 1024)
				system.Memory.PageTablesBytes = uint64(osSnapshot.Memory.PageTables * 1024)
				system.Memory.SlabBytes = uint64(osSnapshot.Memory.Slab * 1024)
				system.Memory.SwapTotalBytes = uint64(osSnapshot.Swap.Total) * 1024
				system.Memory.SwapUsedBytes = uint64(osSnapshot.Swap.Total-osSnapshot.Swap.Free) * 1024
				system.Memory.TotalBytes = uint64(osSnapshot.Memory.Total * 1024)
				system.Memory.WritebackBytes = uint64(osSnapshot.Memory.Writeback * 1024)

				system.NetworkStats = make(state.NetworkStatsMap)
				for _, networkIf := range osSnapshot.Network {
					// Practically this always has one entry, and oddly enough we don't have
					// the throughput numbers on a per interface basis...
					system.NetworkStats[networkIf.Interface] = state.NetworkStats{
						DiffedOnInput: true,
						DiffedValues: &state.DiffedNetworkStats{
							ReceiveThroughputBytesPerSecond:  uint64(cloudWatchReader.GetRdsIntMetric("NetworkReceiveThroughput", "Bytes/Second")),
							TransmitThroughputBytesPerSecond: uint64(cloudWatchReader.GetRdsIntMetric("NetworkTransmitThroughput", "Bytes/Second")),
						},
					}
				}

				for _, disk := range osSnapshot.DiskIO {
					// "The rdsdev device relates to the /rdsdbdata file system, where all database files and logs are stored"
					// Source: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Monitoring.OS.html
					if disk.Device == "rdsdev" {
						system.DiskStats["default"].DiffedValues.UtilizationPercent = float64(disk.Util)
					}
				}

				system.DataDirectoryPartition = "/rdsdbdata"
				system.DiskPartitions = make(state.DiskPartitionMap)
				for _, diskPartition := range osSnapshot.FileSystems {
					totalBytes := uint64(diskPartition.Total * 1024)
					if isAurora {
						totalBytes = AuroraMaxStorage
					}
					system.DiskPartitions[diskPartition.MountPoint] = state.DiskPartition{
						DiskName:      "default",
						PartitionName: diskPartition.Name,
						UsedBytes:     uint64(diskPartition.Used * 1024),
						TotalBytes:    totalBytes,
					}
				}
			}
		}
	} else {
		system.CPUStats = make(state.CPUStatisticMap)
		system.CPUStats["all"] = state.CPUStatistic{
			DiffedOnInput: true,
			DiffedValues: &state.DiffedSystemCPUStats{
				UserPercent: cloudWatchReader.GetRdsFloatMetric("CPUUtilization", "Percent"),
			},
		}

		system.NetworkStats = make(state.NetworkStatsMap)
		system.NetworkStats["default"] = state.NetworkStats{
			DiffedOnInput: true,
			DiffedValues: &state.DiffedNetworkStats{
				ReceiveThroughputBytesPerSecond:  uint64(cloudWatchReader.GetRdsIntMetric("NetworkReceiveThroughput", "Bytes/Second")),
				TransmitThroughputBytesPerSecond: uint64(cloudWatchReader.GetRdsIntMetric("NetworkTransmitThroughput", "Bytes/Second")),
			},
		}

		system.Memory.FreeBytes = uint64(cloudWatchReader.GetRdsIntMetric("FreeableMemory", "Bytes"))
		system.Memory.SwapUsedBytes = uint64(cloudWatchReader.GetRdsIntMetric("SwapUsage", "Bytes"))

		var bytesTotal, bytesFree int64
		if instance.AllocatedStorage != nil {
			bytesTotal = *instance.AllocatedStorage * 1024 * 1024 * 1024
			bytesFree = cloudWatchReader.GetRdsIntMetric("FreeStorageSpace", "Bytes")

			totalBytes := uint64(bytesTotal)
			if isAurora {
				totalBytes = AuroraMaxStorage
			}

			system.DiskPartitions = make(state.DiskPartitionMap)
			system.DiskPartitions["/"] = state.DiskPartition{
				DiskName:   "default",
				UsedBytes:  uint64(bytesTotal - bytesFree),
				TotalBytes: totalBytes,
			}
		}
	}

	server.SelfTest.MarkCollectionAspectOk(state.CollectionAspectSystemStats)

	return
}
