package systemstats

import (
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/guregu/null.v2"

	//"github.com/aws/aws-sdk-go/aws/request"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

// AmazonRdsInfo - Additional information for Amazon RDS systems
type AmazonRdsInfo struct {
	Region                    *string `json:"region"`
	InstanceClass             *string `json:"instance_class"`    // e.g. "db.m3.xlarge"
	InstanceID                *string `json:"instance_id"`       // e.g. "my-database"
	Status                    *string `json:"status"`            // e.g. "available"
	AvailabilityZone          *string `json:"availability_zone"` // e.g. "us-east-1a"
	PubliclyAccessible        *bool   `json:"publicly_accessible"`
	MultiAZ                   *bool   `json:"multi_az"`
	SecondaryAvailabilityZone *string `json:"secondary_availability_zone"` // e.g. "us-east-1c"
	CACertificate             *string `json:"ca_certificate"`              // e.g. "rds-ca-2015"

	AutoMinorVersionUpgrade    *bool   `json:"auto_minor_version_upgrade"`
	PreferredMaintenanceWindow *string `json:"preferred_maintenance_window"`

	LatestRestorableTime  *time.Time `json:"latest_restorable_time"`
	PreferredBackupWindow *string    `json:"preferred_backup_window"`
	BackupRetentionPeriod *int64     `json:"backup_retention_period"` // e.g. 7 (in number of days)

	MasterUsername *string    `json:"master_username"`
	InitialDbName  *string    `json:"initial_db_name"`
	CreatedAt      *time.Time `json:"created_at"`

	StorageProvisionedIops *int64  `json:"storage_provisioned_iops"`
	StorageEncrypted       *bool   `json:"storage_encrypted"`
	StorageType            *string `json:"storage_type"`

	ParameterApplyStatus string `json:"parameter_apply_status"` // e.g. pending-reboot
	ParameterPgssEnabled bool   `json:"parameter_pgss_enabled"`

	OsSnapshot *RdsOsSnapshot `json:"os_snapshot"`

	// ---

	// If the DB instance is a member of a DB cluster, contains the name of the
	// DB cluster that the DB instance is a member of.
	//DBClusterIdentifier *string `type:"string"`

	// Contains one or more identifiers of the Read Replicas associated with this
	// DB instance.
	//ReadReplicaDBInstanceIdentifiers []*string `locationNameList:"ReadReplicaDBInstanceIdentifier" type:"list"`

	// Contains the identifier of the source DB instance if this DB instance is
	// a Read Replica.
	//ReadReplicaSourceDBInstanceIdentifier *string `type:"string"`

	// The status of a Read Replica. If the instance is not a Read Replica, this
	// will be blank.
	//StatusInfos []*DBInstanceStatusInfo `locationNameList:"DBInstanceStatusInfo" type:"list"`
}

// http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_Monitoring.html

type RdsOsSnapshot struct {
	Engine             string  `json:"engine"`             // The database engine for the DB instance.
	InstanceID         string  `json:"instanceID"`         // The DB instance identifier.
	InstanceResourceID string  `json:"instanceResourceID"` // A region-unique, immutable identifier for the DB instance, also used as the log stream identifier.
	Timestamp          string  `json:"timestamp"`          // The time at which the metrics were taken.
	Version            float32 `json:"version"`            // The version of the OS metrics' stream JSON format.
	Uptime             string  `json:"uptime"`             // The amount of time that the DB instance has been active.
	NumVCPUs           int32   `json:"numVCPUs"`           // The number of virtual CPUs for the DB instance.

	CPUUtilization    RdsOsCPUUtilization     `json:"cpuUtilization"`
	LoadAverageMinute RdsOsLoadAverageMinute  `json:"loadAverageMinute"`
	Memory            RdsOsMemory             `json:"memory"`
	Tasks             RdsOsTasks              `json:"tasks"`
	Swap              RdsOsSwap               `json:"swap"`
	Network           []RdsOsNetworkInterface `json:"network"`
	DiskIO            []RdsOsDiskIO           `json:"diskIO"`
	FileSystems       []RdsOsFileSystem       `json:"fileSys"`

	// Skip this for now to reduce output size
	// ProcessList []RdsOsProcess `json:"processList"`
}

type RdsOsCPUUtilization struct {
	Guest  float32 `json:"guest"`  // The percentage of CPU in use by guest programs.
	Irq    float32 `json:"irq"`    // The percentage of CPU in use by software interrupts.
	System float32 `json:"system"` // The percentage of CPU in use by the kernel.
	Wait   float32 `json:"wait"`   // The percentage of CPU unused while waiting for I/O access.
	Idle   float32 `json:"idle"`   // The percentage of CPU that is idle.
	User   float32 `json:"user"`   // The percentage of CPU in use by user programs.
	Total  float32 `json:"total"`  // The total percentage of the CPU in use. This value excludes the nice value.
	Steal  float32 `json:"steal"`  // The percentage of CPU in use by other virtual machines.
	Nice   float32 `json:"nice"`   // The percentage of CPU in use by programs running at lowest priority.
}

type RdsOsLoadAverageMinute struct {
	Fifteen float32 `json:"fifteen"` // The number of processes requesting CPU time over the last 15 minutes.
	Five    float32 `json:"five"`    // The number of processes requesting CPU time over the last 5 minutes.
	One     float32 `json:"one"`     // The number of processes requesting CPU time over the last minute.
}

type RdsOsMemory struct {
	Writeback      int64 `json:"writeback"`      // The amount of dirty pages in RAM that are still being written to the backing storage, in kilobytes.
	HugePagesFree  int64 `json:"hugePagesFree"`  // The number of free huge pages. Huge pages are a feature of the Linux kernel.
	HugePagesRsvd  int64 `json:"hugePagesRsvd"`  // The number of committed huge pages.
	HugePagesSurp  int64 `json:"hugePagesSurp"`  // The number of available surplus huge pages over the total.
	Cached         int64 `json:"cached"`         // The amount of memory used for caching file system–based I/O.
	HugePagesSize  int64 `json:"hugePagesSize"`  // The size for each huge pages unit, in kilobytes.
	Free           int64 `json:"free"`           // The amount of unassigned memory, in kilobytes.
	HugePagesTotal int64 `json:"hugePagesTotal"` // The total number of huge pages for the system.
	Inactive       int64 `json:"inactive"`       // The amount of least-frequently used memory pages, in kilobytes.
	PageTables     int64 `json:"pageTables"`     // The amount of memory used by page tables, in kilobytes.
	Dirty          int64 `json:"dirty"`          // The amount of memory pages in RAM that have been modified but not written to their related data block in storage, in kilobytes.
	Mapped         int64 `json:"mapped"`         // The total amount of file-system contents that is memory mapped inside a process address space, in kilobytes.
	Active         int64 `json:"active"`         // The amount of assigned memory, in kilobytes.
	Total          int64 `json:"total"`          // The total amount of memory, in kilobytes.
	Slab           int64 `json:"slab"`           // The amount of reusable kernel data structures, in kilobytes.
	Buffers        int64 `json:"buffers"`        // The amount of memory used for buffering I/O requests prior to writing to the storage device, in kilobytes.
}

type RdsOsTasks struct {
	Sleeping int64 `json:"sleeping"` // The number of tasks that are sleeping.
	Zombie   int64 `json:"zombie"`   // The number of child tasks that are inactive with an active parent task.
	Running  int64 `json:"running"`  // The number of tasks that are running.
	Stopped  int64 `json:"stopped"`  // The number of tasks that are stopped.
	Total    int64 `json:"total"`    // The total number of tasks.
	Blocked  int64 `json:"blocked"`  // The number of tasks that are blocked.
}

type RdsOsSwap struct {
	Cached int64 `json:"cached"` // The amount of swap memory, in kilobytes, used as cache memory.
	Total  int64 `json:"total"`  // The total amount of swap memory available, in kilobytes.
	Free   int64 `json:"free"`   // The total amount of swap memory free, in kilobytes.
}

type RdsOsNetworkInterface struct {
	Interface string  `json:"interface"` // The identifier for the network interface being used for the DB instance.
	Rx        float64 `json:"rx"`        // The number of packets received.
	Tx        float64 `json:"tx"`        // The number of packets uploaded.
}

type RdsOsDiskIO struct {
	WriteKbPS   float32 `json:"writeKbPS"`   // The number of kilobytes written per second.
	ReadIOsPS   float32 `json:"readIOsPS"`   // The number of read operations per second.
	Await       float32 `json:"await"`       // The number of milliseconds required to respond to requests, including queue time and service time.
	ReadKbPS    float32 `json:"readKbPS"`    // The number of kilobytes read per second.
	RrqmPS      float32 `json:"rrqmPS"`      // The number of merged read requests queued per second.
	Util        float32 `json:"util"`        // The percentage of CPU time during which requests were issued.
	AvgQueueLen float32 `json:"avgQueueLen"` // The number of requests waiting in the I/O device's queue.
	Tps         float32 `json:"tps"`         // The number of I/O transactions per second.
	ReadKb      float32 `json:"readKb"`      // The total number of kilobytes read.
	Device      string  `json:"device"`      // The identifier of the disk device in use.
	WriteKb     float32 `json:"writeKb"`     // The total number of kilobytes written.
	AvgReqSz    float32 `json:"avgReqSz"`    // The average request size, in kilobytes.
	WrqmPS      float32 `json:"wrqmPS"`      // The number of merged write requests queued per second.
	WriteIOsPS  float32 `json:"writeIOsPS"`  // The number of write operations per second.
}

type RdsOsFileSystem struct {
	Used            int64   `json:"used"`            // The amount of disk space used by files in the file system, in kilobytes.
	Name            string  `json:"name"`            // The name of the file system.
	UsedFiles       int64   `json:"usedFiles"`       // The number of files in the file system.
	UsedFilePercent float32 `json:"usedFilePercent"` // The percentage of available files in use.
	MaxFiles        int64   `json:"maxFiles"`        // The maximum number of files that can be created for the file system.
	MountPoint      string  `json:"mountPoint"`      // The path to the file system.
	Total           int64   `json:"total"`           // The total number of disk space available for the file system, in kilobytes.
	UsedPercent     float32 `json:"usedPercent"`     // The percentage of the file-system disk space in use.
}

type RdsOsProcess struct {
	Vss          int64   `json:"vss"`          // The amount of virtual memory allocated to the process, in kilobytes.
	Name         string  `json:"name"`         // The name of the process.
	Tgid         int64   `json:"tgid"`         // The thread group identifier, which is a number representing the process ID to which a thread belongs. This identifier is used to group threads from the same process.
	ParentID     int64   `json:"parentID"`     // The process identifier for the parent process of the process.
	MemoryUsedPc float32 `json:"memoryUsedPc"` // The percentage of memory used by the process.
	CPUUsedPc    float32 `json:"cpuUsedPc"`    // The percentage of CPU used by the process.
	ID           int64   `json:"id"`           // The identifier of the process.
	Rss          int64   `json:"rss"`          // The amount of RAM allocated to the process, in kilobytes.
}

// GetFromAmazonRds - Gets system information about an Amazon RDS instance
func getFromAmazonRds(config config.DatabaseConfig) (system *SystemSnapshot) {
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

	system = &SystemSnapshot{
		SystemType: AmazonRdsSystem,
	}

	systemInfo := &AmazonRdsInfo{
		Region:                    &config.AwsRegion,
		InstanceClass:             instance.DBInstanceClass,
		InstanceID:                instance.DBInstanceIdentifier,
		Status:                    instance.DBInstanceStatus,
		AvailabilityZone:          instance.AvailabilityZone,
		PubliclyAccessible:        instance.PubliclyAccessible,
		MultiAZ:                   instance.MultiAZ,
		SecondaryAvailabilityZone: instance.SecondaryAvailabilityZone,
		CACertificate:             instance.CACertificateIdentifier,

		AutoMinorVersionUpgrade:    instance.AutoMinorVersionUpgrade,
		PreferredMaintenanceWindow: instance.PreferredMaintenanceWindow,

		LatestRestorableTime:  instance.LatestRestorableTime,
		PreferredBackupWindow: instance.PreferredBackupWindow,
		BackupRetentionPeriod: instance.BackupRetentionPeriod,

		MasterUsername: instance.MasterUsername,
		InitialDbName:  instance.DBName,
		CreatedAt:      instance.InstanceCreateTime,

		StorageProvisionedIops: instance.Iops,
		StorageEncrypted:       instance.StorageEncrypted,
		StorageType:            instance.StorageType,
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

	system.Network = &Network{
		ReceiveThroughput:  util.GetRdsIntMetric(dbInstanceID, "NetworkReceiveThroughput", "Bytes/Second", sess),
		TransmitThroughput: util.GetRdsIntMetric(dbInstanceID, "NetworkTransmitThroughput", "Bytes/Second", sess),
	}

	storage := Storage{
		BytesAvailable: util.GetRdsIntMetric(dbInstanceID, "FreeStorageSpace", "Bytes", sess),
		Perfdata: StoragePerfdata{
			Version:         1,
			ReadIops:        util.GetRdsIntMetric(dbInstanceID, "ReadIOPS", "Count/Second", sess),
			WriteIops:       util.GetRdsIntMetric(dbInstanceID, "WriteIOPS", "Count/Second", sess),
			ReadThroughput:  util.GetRdsIntMetric(dbInstanceID, "ReadThroughput", "Bytes/Second", sess),
			WriteThroughput: util.GetRdsIntMetric(dbInstanceID, "WriteThroughput", "Bytes/Second", sess),
			IopsInProgress:  util.GetRdsIntMetric(dbInstanceID, "DiskQueueDepth", "Count", sess),
			ReadLatency:     util.GetRdsFloatMetric(dbInstanceID, "ReadLatency", "Seconds", sess),
			WriteLatency:    util.GetRdsFloatMetric(dbInstanceID, "WriteLatency", "Seconds", sess),
		},
	}

	var bytesTotal int64
	if instance.AllocatedStorage != nil {
		bytesTotal = *instance.AllocatedStorage * 1024 * 1024 * 1024
		storage.BytesTotal = &bytesTotal
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
				var osSnapshot RdsOsSnapshot
				err = json.Unmarshal([]byte(*str), &osSnapshot)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					return
				}

				// Technically these are not msec but percentages, so we multiply by the number of milliseconds in a minute (our standard measurement)
				system.CPU.BusyTimesGuestMsec = null.IntFrom(int64(osSnapshot.CPUUtilization.Guest * 60000))
				system.CPU.BusyTimesGuestNiceMsec = null.IntFrom(0)
				system.CPU.BusyTimesIdleMsec = null.IntFrom(int64(osSnapshot.CPUUtilization.Idle * 60000))
				system.CPU.BusyTimesSoftirqMsec = null.IntFrom(int64(osSnapshot.CPUUtilization.Irq * 60000))
				system.CPU.BusyTimesIrqMsec = null.IntFrom(0)
				system.CPU.BusyTimesIowaitMsec = null.IntFrom(int64(osSnapshot.CPUUtilization.Wait * 60000))
				system.CPU.BusyTimesSystemMsec = null.IntFrom(int64(osSnapshot.CPUUtilization.System * 60000))
				system.CPU.BusyTimesUserMsec = null.IntFrom(int64(osSnapshot.CPUUtilization.User * 60000))
				system.CPU.BusyTimesStealMsec = null.IntFrom(int64(osSnapshot.CPUUtilization.Steal * 60000))
				system.CPU.BusyTimesNiceMsec = null.IntFrom(int64(osSnapshot.CPUUtilization.Nice * 60000))

				system.CPU.HardwareSockets = null.IntFrom(1)
				system.CPU.HardwareCoresPerSocket = null.IntFrom(int64(osSnapshot.NumVCPUs))

				system.Scheduler.Loadavg1min = null.FloatFrom(float64(osSnapshot.LoadAverageMinute.One))
				system.Scheduler.Loadavg5min = null.FloatFrom(float64(osSnapshot.LoadAverageMinute.Five))
				system.Scheduler.Loadavg15min = null.FloatFrom(float64(osSnapshot.LoadAverageMinute.Fifteen))

				system.Scheduler.ProcsRunning = null.IntFrom(osSnapshot.Tasks.Running)
				system.Scheduler.ProcsBlocked = null.IntFrom(osSnapshot.Tasks.Blocked)

				system.Memory.ApplicationsBytes = null.IntFrom((osSnapshot.Memory.Total - osSnapshot.Memory.Free - osSnapshot.Memory.Buffers - osSnapshot.Memory.Cached) * 1024)
				system.Memory.BuffersBytes = null.IntFrom(osSnapshot.Memory.Buffers * 1024)
				system.Memory.DirtyBytes = null.IntFrom(osSnapshot.Memory.Dirty * 1024)
				system.Memory.FreeBytes = null.IntFrom(osSnapshot.Memory.Free * 1024)
				system.Memory.PagecacheBytes = null.IntFrom(osSnapshot.Memory.Cached * 1024)
				system.Memory.SwapFreeBytes = null.IntFrom(osSnapshot.Swap.Free * 1024)
				system.Memory.SwapTotalBytes = null.IntFrom(osSnapshot.Swap.Total * 1024)
				system.Memory.TotalBytes = null.IntFrom(osSnapshot.Memory.Total * 1024)
				system.Memory.WritebackBytes = null.IntFrom(osSnapshot.Memory.Writeback * 1024)
				system.Memory.ActiveBytes = null.IntFrom(osSnapshot.Memory.Active * 1024)

				system.Storage[0].Perfdata.AvgReqSize = null.IntFrom(int64(osSnapshot.DiskIO[0].AvgReqSz * 1024))

				systemInfo.OsSnapshot = &osSnapshot
			}
		}
	} else {
		system.Memory.FreeBytes = null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "FreeableMemory", "Bytes", sess))
		system.Memory.SwapTotalBytes = null.IntFromPtr(util.GetRdsIntMetric(dbInstanceID, "SwapUsage", "Bytes", sess))
		system.Memory.SwapFreeBytes = null.IntFrom(0)
	}

	return
}
