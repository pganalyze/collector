package systemstats

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	//"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/lfittl/pganalyze-collector-next/config"
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
func getFromAmazonRds(config config.Config) (system *SystemSnapshot) {
	creds := credentials.NewStaticCredentials(config.AwsAccessKeyId, config.AwsSecretAccessKey, "")

	sess := session.New(&aws.Config{Credentials: creds, Region: aws.String(config.AwsRegion)})
	//sess.Handlers.Send.PushFront(func(r *request.Request) {
	// Log every request made and its payload
	//  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
	//})

	rdsSvc := rds.New(sess)

	instance, err := findInstance(config, sess)

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

	pgssParam, _ := getParameter(group, "shared_preload_libraries", rdsSvc)

	systemInfo.ParameterPgssEnabled = pgssParam != nil && *pgssParam.ParameterValue == "pg_stat_statements"
	systemInfo.ParameterApplyStatus = *group.ParameterApplyStatus

	system.SystemInfo = systemInfo

	// Not fetched right now:
	// - DatabaseConnections
	// - TransactionLogsDiskUsage

	dbInstanceID := *instance.DBInstanceIdentifier

	system.CPU.Utilization = getFloatMetric(dbInstanceID, "CPUUtilization", "Percent", sess)

	system.Network = &Network{
		ReceiveThroughput:  getIntMetric(dbInstanceID, "NetworkReceiveThroughput", "Bytes/Second", sess),
		TransmitThroughput: getIntMetric(dbInstanceID, "NetworkTransmitThroughput", "Bytes/Second", sess),
	}

	system.Memory.FreeBytes = getIntMetric(dbInstanceID, "FreeableMemory", "Bytes", sess)
	system.Memory.SwapTotalBytes = getIntMetric(dbInstanceID, "SwapUsage", "Bytes", sess)

	var swapFree int64
	system.Memory.SwapFreeBytes = &swapFree

	storage := Storage{
		BytesAvailable: getIntMetric(dbInstanceID, "FreeStorageSpace", "Bytes", sess),
		Perfdata: StoragePerfdata{
			Version:         1,
			ReadIops:        getIntMetric(dbInstanceID, "ReadIOPS", "Count/Second", sess),
			WriteIops:       getIntMetric(dbInstanceID, "WriteIOPS", "Count/Second", sess),
			ReadThroughput:  getIntMetric(dbInstanceID, "ReadThroughput", "Bytes/Second", sess),
			WriteThroughput: getIntMetric(dbInstanceID, "WriteThroughput", "Bytes/Second", sess),
			IopsInProgress:  getIntMetric(dbInstanceID, "DiskQueueDepth", "Count", sess),
			ReadLatency:     getFloatMetric(dbInstanceID, "ReadLatency", "Seconds", sess),
			WriteLatency:    getFloatMetric(dbInstanceID, "WriteLatency", "Seconds", sess),
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

func findInstance(config config.Config, sess *session.Session) (instance *rds.DBInstance, err error) {
	var resp *rds.DescribeDBInstancesOutput

	svc := rds.New(sess)

	if config.AwsDbInstanceId != "" {
		params := &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(config.AwsDbInstanceId),
		}

		resp, err = svc.DescribeDBInstances(params)

		if err == nil && len(resp.DBInstances) >= 1 {
			instance = resp.DBInstances[0]
		}

		return
	}

	params := &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int64(100),
	}

	resp, err = svc.DescribeDBInstances(params)
	if err != nil {
		return
	}

	for _, instance = range resp.DBInstances {
		host := instance.Endpoint.Address
		port := instance.Endpoint.Port
		if host != nil && port != nil && *host == config.GetDbHost() && *port == int64(config.GetDbPort()) {
			return
		}
	}

	instance = nil
	return
}

func getParameter(group *rds.DBParameterGroupStatus, name string, svc *rds.RDS) (parameter *rds.Parameter, err error) {
	var resp *rds.DescribeDBParametersOutput

	params := &rds.DescribeDBParametersInput{
		DBParameterGroupName: aws.String(*group.DBParameterGroupName),
	}

	for {
		resp, err = svc.DescribeDBParameters(params)
		if err != nil {
			return
		}

		for _, parameter = range resp.Parameters {
			if parameter.ParameterName != nil && *parameter.ParameterName == name {
				return
			}
		}

		params.Marker = resp.Marker

		if params.Marker == nil {
			break
		}
	}

	parameter = nil
	return
}

// GetIntMetric - Gets an integer value from Cloudwatch
func getIntMetric(instance string, metricName string, unit string, sess *session.Session) *int64 {
	value := getFloatMetric(instance, metricName, unit, sess)
	if value == nil {
		return nil
	}
	var valueInt = int64(*value)
	return &valueInt
}

// GetFloatMetric - Gets a float value from Cloudwatch
func getFloatMetric(instance string, metricName string, unit string, sess *session.Session) *float64 {
	svc := cloudwatch.New(sess)

	params := &cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String(metricName),
		Namespace:  aws.String("AWS/RDS"),
		Period:     aws.Int64(60),
		StartTime:  aws.Time(time.Now().Add(-10 * time.Minute)),
		Unit:       aws.String(unit),
		Statistics: []*string{
			aws.String("Average"),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("DBInstanceIdentifier"),
				Value: aws.String(instance),
			},
		},
	}
	resp, err := svc.GetMetricStatistics(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return nil
	}

	if len(resp.Datapoints) == 0 {
		return nil
	}

	return resp.Datapoints[0].Average
}
