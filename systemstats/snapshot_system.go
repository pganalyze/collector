package systemstats

import "time"

type SnapshotSystem struct {
	Storage Storage  `json:"storage"`
	Memory  Memory   `json:"memory"`
	Rds     *RdsInfo `json:"rds"`
}

type RdsInfo struct {
	InstanceClass             *string `json:"instance_class"`    // e.g. "db.m3.xlarge"
	InstanceId                *string `json:"instance_id"`       // e.g. "my-database"
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
