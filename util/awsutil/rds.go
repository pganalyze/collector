package awsutil

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

func findRdsClusterByIdentifier(clusterIdentifier string, svc *rds.RDS) (*rds.DBCluster, error) {
	params := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
	}

	resp, err := svc.DescribeDBClusters(params)
	if err != nil {
		return nil, err
	}
	if len(resp.DBClusters) == 0 {
		return nil, fmt.Errorf("Unexpected empty result set for DescribeDBClusters with DBClusterIdentifier = \"%s\"", clusterIdentifier)
	}

	return resp.DBClusters[0], nil
}

func findRdsInstanceByIdentifier(instanceIdentifier string, svc *rds.RDS) (*rds.DBInstance, error) {
	params := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceIdentifier),
	}

	resp, err := svc.DescribeDBInstances(params)
	if err != nil {
		return nil, err
	}

	if len(resp.DBInstances) == 0 {
		return nil, fmt.Errorf("Unexpected empty result set for DescribeDBInstances with DBInstanceIdentifier = \"%s\"", instanceIdentifier)
	}

	return resp.DBInstances[0], nil
}

func findRdsInstanceByHostAndPort(host string, port int64, svc *rds.RDS) (*rds.DBInstance, error) {
	params := &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int64(100),
	}

	resp, err := svc.DescribeDBInstances(params)
	if err != nil {
		return nil, err
	}

	for _, instance := range resp.DBInstances {
		instanceHost := instance.Endpoint.Address
		instancePort := instance.Endpoint.Port
		if instanceHost != nil && instancePort != nil && *instanceHost == host && *instancePort == port {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("Failed to find RDS instance using endpoint-based search for host \"%s\" and port %d", host, port)
}

func FindRdsInstance(config config.ServerConfig, sess *session.Session) (*rds.DBInstance, error) {
	svc := rds.New(sess)

	if config.AwsDbInstanceID != "" {
		return findRdsInstanceByIdentifier(config.AwsDbInstanceID, svc)
	}

	if config.AwsDbClusterID != "" {
		cluster, err := findRdsClusterByIdentifier(config.AwsDbClusterID, svc)
		if err != nil {
			return nil, err
		}

		instanceID := ""
		for _, clusterMember := range cluster.DBClusterMembers {
			if (config.AwsDbClusterReadonly && *clusterMember.IsClusterWriter) ||
				(!config.AwsDbClusterReadonly && !*clusterMember.IsClusterWriter) {
				continue
			}
			if instanceID == "" {
				instanceID = *clusterMember.DBInstanceIdentifier
			} else if config.AwsDbClusterReadonly {
				return nil, fmt.Errorf("Found more than one reader to monitor for read-only cluster \"%s\" (HINT: use specific instance IDs instead)", config.AwsDbClusterID)
			} else {
				return nil, fmt.Errorf("Unexpected multiple writers for cluster \"%s\"", config.AwsDbClusterID)
			}
		}

		if instanceID == "" {
			return nil, fmt.Errorf("Could not locate usable instance ID for cluster \"%s\" (readonly = %t)", config.AwsDbClusterID, config.AwsDbClusterReadonly)
		}

		return findRdsInstanceByIdentifier(instanceID, svc)
	}

	// If neither instance ID or cluster ID were specified, but we still have
	// an RDS system type, attempt to find the instance based on the hostname
	// (this is a long shot, but there are some cases where this helps)
	return findRdsInstanceByHostAndPort(config.GetDbHost(), int64(config.GetDbPort()), svc)
}

func GetRdsParameter(group *rds.DBParameterGroupStatus, name string, svc *rds.RDS) (parameter *rds.Parameter, err error) {
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

type RdsCloudWatchReader struct {
	svc      *cloudwatch.CloudWatch
	instance string
	logger   *util.Logger
}

func NewRdsCloudWatchReader(sess *session.Session, logger *util.Logger, instance string) RdsCloudWatchReader {
	return RdsCloudWatchReader{svc: cloudwatch.New(sess), instance: instance, logger: logger}
}

// GetRdsIntMetric - Gets an integer value from Cloudwatch
func (reader RdsCloudWatchReader) GetRdsIntMetric(metricName string, unit string) int64 {
	return int64(reader.GetRdsFloatMetric(metricName, unit))
}

// GetRdsFloatMetric - Gets a float value from Cloudwatch
func (reader RdsCloudWatchReader) GetRdsFloatMetric(metricName string, unit string) float64 {
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
				Value: aws.String(reader.instance),
			},
		},
	}
	resp, err := reader.svc.GetMetricStatistics(params)

	if err != nil {
		reader.logger.PrintVerbose(err.Error())
		return 0.0
	}

	if len(resp.Datapoints) == 0 {
		return 0.0
	}

	val := resp.Datapoints[0].Average
	if val != nil {
		return *resp.Datapoints[0].Average
	}

	return 0.0
}
