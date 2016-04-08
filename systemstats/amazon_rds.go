package systemstats

import (
	"fmt"
	"time"

	//"github.com/aws/aws-sdk-go/aws/request"

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

	system.Memory.FreeBytes = util.GetRdsIntMetric(dbInstanceID, "FreeableMemory", "Bytes", sess)
	system.Memory.SwapTotalBytes = util.GetRdsIntMetric(dbInstanceID, "SwapUsage", "Bytes", sess)

	var swapFree int64
	system.Memory.SwapFreeBytes = &swapFree

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

	return
}
