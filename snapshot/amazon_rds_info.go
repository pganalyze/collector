//go:generate msgp

package snapshot

// AmazonRdsInfo - Additional information for Amazon RDS systems
type AmazonRdsInfo struct {
	Region                    NullableString `msg:"region"`
	InstanceClass             NullableString `msg:"instance_class"`    // e.g. "db.m3.xlarge"
	InstanceID                NullableString `msg:"instance_id"`       // e.g. "my-database"
	Status                    NullableString `msg:"status"`            // e.g. "available"
	AvailabilityZone          NullableString `msg:"availability_zone"` // e.g. "us-east-1a"
	PubliclyAccessible        NullableBool   `msg:"publicly_accessible"`
	MultiAZ                   NullableBool   `msg:"multi_az"`
	SecondaryAvailabilityZone NullableString `msg:"secondary_availability_zone"` // e.g. "us-east-1c"
	CACertificate             NullableString `msg:"ca_certificate"`              // e.g. "rds-ca-2015"

	AutoMinorVersionUpgrade    NullableBool   `msg:"auto_minor_version_upgrade"`
	PreferredMaintenanceWindow NullableString `msg:"preferred_maintenance_window"`

	LatestRestorableTime  NullableUnixTimestamp `msg:"latest_restorable_time"`
	PreferredBackupWindow NullableString        `msg:"preferred_backup_window"`
	BackupRetentionPeriod NullableInt           `msg:"backup_retention_period"` // e.g. 7 (in number of days)

	MasterUsername NullableString        `msg:"master_username"`
	InitialDbName  NullableString        `msg:"initial_db_name"`
	CreatedAt      NullableUnixTimestamp `msg:"created_at"`

	StorageProvisionedIops NullableInt    `msg:"storage_provisioned_iops"`
	StorageEncrypted       NullableBool   `msg:"storage_encrypted"`
	StorageType            NullableString `msg:"storage_type"`

	ParameterApplyStatus string `msg:"parameter_apply_status"` // e.g. pending-reboot
	ParameterPgssEnabled bool   `msg:"parameter_pgss_enabled"`

	OsSnapshot *RdsOsSnapshot `msg:"os_snapshot"`

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
	Engine             string  `msg:"engine"`             // The database engine for the DB instance.
	InstanceID         string  `msg:"instanceID"`         // The DB instance identifier.
	InstanceResourceID string  `msg:"instanceResourceID"` // A region-unique, immutable identifier for the DB instance, also used as the log stream identifier.
	Timestamp          string  `msg:"timestamp"`          // The time at which the metrics were taken.
	Version            float32 `msg:"version"`            // The version of the OS metrics' stream JSON format.
	Uptime             string  `msg:"uptime"`             // The amount of time that the DB instance has been active.
	NumVCPUs           int32   `msg:"numVCPUs"`           // The number of virtual CPUs for the DB instance.

	CPUUtilization    RdsOsCPUUtilization     `msg:"cpuUtilization"`
	LoadAverageMinute RdsOsLoadAverageMinute  `msg:"loadAverageMinute"`
	Memory            RdsOsMemory             `msg:"memory"`
	Tasks             RdsOsTasks              `msg:"tasks"`
	Swap              RdsOsSwap               `msg:"swap"`
	Network           []RdsOsNetworkInterface `msg:"network"`
	DiskIO            []RdsOsDiskIO           `msg:"diskIO"`
	FileSystems       []RdsOsFileSystem       `msg:"fileSys"`

	// Skip this for now to reduce output size
	// ProcessList []RdsOsProcess `msg:"processList"`
}

type RdsOsCPUUtilization struct {
	Guest  float32 `msg:"guest"`  // The percentage of CPU in use by guest programs.
	Irq    float32 `msg:"irq"`    // The percentage of CPU in use by software interrupts.
	System float32 `msg:"system"` // The percentage of CPU in use by the kernel.
	Wait   float32 `msg:"wait"`   // The percentage of CPU unused while waiting for I/O access.
	Idle   float32 `msg:"idle"`   // The percentage of CPU that is idle.
	User   float32 `msg:"user"`   // The percentage of CPU in use by user programs.
	Total  float32 `msg:"total"`  // The total percentage of the CPU in use. This value excludes the nice value.
	Steal  float32 `msg:"steal"`  // The percentage of CPU in use by other virtual machines.
	Nice   float32 `msg:"nice"`   // The percentage of CPU in use by programs running at lowest priority.
}

type RdsOsLoadAverageMinute struct {
	Fifteen float32 `msg:"fifteen"` // The number of processes requesting CPU time over the last 15 minutes.
	Five    float32 `msg:"five"`    // The number of processes requesting CPU time over the last 5 minutes.
	One     float32 `msg:"one"`     // The number of processes requesting CPU time over the last minute.
}

type RdsOsMemory struct {
	Writeback      int64 `msg:"writeback"`      // The amount of dirty pages in RAM that are still being written to the backing storage, in kilobytes.
	HugePagesFree  int64 `msg:"hugePagesFree"`  // The number of free huge pages. Huge pages are a feature of the Linux kernel.
	HugePagesRsvd  int64 `msg:"hugePagesRsvd"`  // The number of committed huge pages.
	HugePagesSurp  int64 `msg:"hugePagesSurp"`  // The number of available surplus huge pages over the total.
	Cached         int64 `msg:"cached"`         // The amount of memory used for caching file system–based I/O.
	HugePagesSize  int64 `msg:"hugePagesSize"`  // The size for each huge pages unit, in kilobytes.
	Free           int64 `msg:"free"`           // The amount of unassigned memory, in kilobytes.
	HugePagesTotal int64 `msg:"hugePagesTotal"` // The total number of huge pages for the system.
	Inactive       int64 `msg:"inactive"`       // The amount of least-frequently used memory pages, in kilobytes.
	PageTables     int64 `msg:"pageTables"`     // The amount of memory used by page tables, in kilobytes.
	Dirty          int64 `msg:"dirty"`          // The amount of memory pages in RAM that have been modified but not written to their related data block in storage, in kilobytes.
	Mapped         int64 `msg:"mapped"`         // The total amount of file-system contents that is memory mapped inside a process address space, in kilobytes.
	Active         int64 `msg:"active"`         // The amount of assigned memory, in kilobytes.
	Total          int64 `msg:"total"`          // The total amount of memory, in kilobytes.
	Slab           int64 `msg:"slab"`           // The amount of reusable kernel data structures, in kilobytes.
	Buffers        int64 `msg:"buffers"`        // The amount of memory used for buffering I/O requests prior to writing to the storage device, in kilobytes.
}

type RdsOsTasks struct {
	Sleeping int64 `msg:"sleeping"` // The number of tasks that are sleeping.
	Zombie   int64 `msg:"zombie"`   // The number of child tasks that are inactive with an active parent task.
	Running  int64 `msg:"running"`  // The number of tasks that are running.
	Stopped  int64 `msg:"stopped"`  // The number of tasks that are stopped.
	Total    int64 `msg:"total"`    // The total number of tasks.
	Blocked  int64 `msg:"blocked"`  // The number of tasks that are blocked.
}

type RdsOsSwap struct {
	Cached int64 `msg:"cached"` // The amount of swap memory, in kilobytes, used as cache memory.
	Total  int64 `msg:"total"`  // The total amount of swap memory available, in kilobytes.
	Free   int64 `msg:"free"`   // The total amount of swap memory free, in kilobytes.
}

type RdsOsNetworkInterface struct {
	Interface string  `msg:"interface"` // The identifier for the network interface being used for the DB instance.
	Rx        float64 `msg:"rx"`        // The number of packets received.
	Tx        float64 `msg:"tx"`        // The number of packets uploaded.
}

type RdsOsDiskIO struct {
	WriteKbPS   float32 `msg:"writeKbPS"`   // The number of kilobytes written per second.
	ReadIOsPS   float32 `msg:"readIOsPS"`   // The number of read operations per second.
	Await       float32 `msg:"await"`       // The number of milliseconds required to respond to requests, including queue time and service time.
	ReadKbPS    float32 `msg:"readKbPS"`    // The number of kilobytes read per second.
	RrqmPS      float32 `msg:"rrqmPS"`      // The number of merged read requests queued per second.
	Util        float32 `msg:"util"`        // The percentage of CPU time during which requests were issued.
	AvgQueueLen float32 `msg:"avgQueueLen"` // The number of requests waiting in the I/O device's queue.
	Tps         float32 `msg:"tps"`         // The number of I/O transactions per second.
	ReadKb      float32 `msg:"readKb"`      // The total number of kilobytes read.
	Device      string  `msg:"device"`      // The identifier of the disk device in use.
	WriteKb     float32 `msg:"writeKb"`     // The total number of kilobytes written.
	AvgReqSz    float32 `msg:"avgReqSz"`    // The average request size, in kilobytes.
	WrqmPS      float32 `msg:"wrqmPS"`      // The number of merged write requests queued per second.
	WriteIOsPS  float32 `msg:"writeIOsPS"`  // The number of write operations per second.
}

type RdsOsFileSystem struct {
	Used            int64   `msg:"used"`            // The amount of disk space used by files in the file system, in kilobytes.
	Name            string  `msg:"name"`            // The name of the file system.
	UsedFiles       int64   `msg:"usedFiles"`       // The number of files in the file system.
	UsedFilePercent float32 `msg:"usedFilePercent"` // The percentage of available files in use.
	MaxFiles        int64   `msg:"maxFiles"`        // The maximum number of files that can be created for the file system.
	MountPoint      string  `msg:"mountPoint"`      // The path to the file system.
	Total           int64   `msg:"total"`           // The total number of disk space available for the file system, in kilobytes.
	UsedPercent     float32 `msg:"usedPercent"`     // The percentage of the file-system disk space in use.
}

type RdsOsProcess struct {
	Vss          int64   `msg:"vss"`          // The amount of virtual memory allocated to the process, in kilobytes.
	Name         string  `msg:"name"`         // The name of the process.
	Tgid         int64   `msg:"tgid"`         // The thread group identifier, which is a number representing the process ID to which a thread belongs. This identifier is used to group threads from the same process.
	ParentID     int64   `msg:"parentID"`     // The process identifier for the parent process of the process.
	MemoryUsedPc float32 `msg:"memoryUsedPc"` // The percentage of memory used by the process.
	CPUUsedPc    float32 `msg:"cpuUsedPc"`    // The percentage of CPU used by the process.
	ID           int64   `msg:"id"`           // The identifier of the process.
	Rss          int64   `msg:"rss"`          // The amount of RAM allocated to the process, in kilobytes.
}
