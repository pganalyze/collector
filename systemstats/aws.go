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

func GetFromAws(config config.Config) (system SnapshotSystem) {
	creds := credentials.NewStaticCredentials(config.AwsAccessKeyId, config.AwsSecretAccessKey, "")

	sess := session.New(&aws.Config{Credentials: creds, Region: aws.String("us-east-1")})
	//sess.Handlers.Send.PushFront(func(r *request.Request) {
	// Log every request made and its payload
	//  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
	//})

	rdsSvc := rds.New(sess)

	instance, err := FindInstance(config, sess)

	if err != nil {
		fmt.Println("Error: %v", err)
		return
	}

	if instance == nil {
		fmt.Println("Could not find RDS instance in AWS, skipping system data")
		return
	}

	system.Rds = &RdsInfo{
		InstanceClass:             instance.DBInstanceClass,
		InstanceId:                instance.DBInstanceIdentifier,
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
		StorageType:            instance.StorageType,
	}

	group := instance.DBParameterGroups[0]

	pgssParam, _ := GetParameter(group, "shared_preload_libraries", rdsSvc)

	system.Rds.ParameterPgssEnabled = pgssParam != nil && *pgssParam.ParameterValue == "pg_stat_statements"
	system.Rds.ParameterApplyStatus = *group.ParameterApplyStatus

	// CPUUtilization
	// DatabaseConnections
	// NetworkReceiveThroughput
	// NetworkTransmitThroughput
	// TransactionLogsDiskUsage

	system.Memory.FreeBytes = GetIntMetric("pganalyze-production", "FreeableMemory", "Bytes", sess)
	system.Memory.SwapTotalBytes = GetIntMetric("pganalyze-production", "SwapUsage", "Bytes", sess)

	var swapFree int64 = 0
	system.Memory.SwapFreeBytes = &swapFree

	system.Storage.Encrypted = instance.StorageEncrypted
	system.Storage.BytesAvailable = GetIntMetric("pganalyze-production", "FreeStorageSpace", "Bytes", sess)
	var bytesTotal int64
	if instance.AllocatedStorage != nil {
		bytesTotal = *instance.AllocatedStorage * 1024 * 1024 * 1024
		system.Storage.BytesTotal = &bytesTotal
	}

	system.Storage.Perfdata.Version = 1
	system.Storage.Perfdata.ReadIops = GetIntMetric("pganalyze-production", "ReadIOPS", "Count/Second", sess)
	system.Storage.Perfdata.WriteIops = GetIntMetric("pganalyze-production", "WriteIOPS", "Count/Second", sess)
	system.Storage.Perfdata.ReadThroughput = GetIntMetric("pganalyze-production", "ReadThroughput", "Bytes/Second", sess)
	system.Storage.Perfdata.WriteThroughput = GetIntMetric("pganalyze-production", "WriteThroughput", "Bytes/Second", sess)
	system.Storage.Perfdata.IopsInProgress = GetIntMetric("pganalyze-production", "DiskQueueDepth", "Count", sess)

	system.Storage.Perfdata.ReadLatency = GetFloatMetric("pganalyze-production", "ReadLatency", "Seconds", sess)
	system.Storage.Perfdata.WriteLatency = GetFloatMetric("pganalyze-production", "WriteLatency", "Seconds", sess)

	return
}

func FindInstance(config config.Config, sess *session.Session) (instance *rds.DBInstance, err error) {
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

func GetParameter(group *rds.DBParameterGroupStatus, name string, svc *rds.RDS) (parameter *rds.Parameter, err error) {
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

func GetIntMetric(instance string, metricName string, unit string, sess *session.Session) *int64 {
	value := GetFloatMetric(instance, metricName, unit, sess)
	if value == nil {
		return nil
	}
	var valueInt = int64(*value)
	return &valueInt
}

func GetFloatMetric(instance string, metricName string, unit string, sess *session.Session) *float64 {
	svc := cloudwatch.New(sess)

	params := &cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String(metricName),
		Namespace:  aws.String("AWS/RDS"),
		Period:     aws.Int64(60),
		StartTime:  aws.Time(time.Now().Add(-1 * time.Minute)),
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
