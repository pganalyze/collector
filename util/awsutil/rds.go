package awsutil

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

func findRdsClusterByIdentifier(ctx context.Context, clusterIdentifier string, client *rds.Client) (*rdstypes.DBCluster, error) {
	resp, err := client.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(clusterIdentifier),
	})
	if err != nil {
		return nil, err
	}
	if len(resp.DBClusters) == 0 {
		return nil, fmt.Errorf("Unexpected empty result set for DescribeDBClusters with DBClusterIdentifier = \"%s\"", clusterIdentifier)
	}
	return &resp.DBClusters[0], nil
}

func findRdsInstanceByIdentifier(ctx context.Context, instanceIdentifier string, client *rds.Client) (*rdstypes.DBInstance, error) {
	resp, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceIdentifier),
	})
	if err != nil {
		return nil, err
	}
	if len(resp.DBInstances) == 0 {
		return nil, fmt.Errorf("Unexpected empty result set for DescribeDBInstances with DBInstanceIdentifier = \"%s\"", instanceIdentifier)
	}
	return &resp.DBInstances[0], nil
}

func findRdsInstanceByHostAndPort(ctx context.Context, host string, port int, client *rds.Client) (*rdstypes.DBInstance, error) {
	resp, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		MaxRecords: aws.Int32(100),
	})
	if err != nil {
		return nil, err
	}
	for i, instance := range resp.DBInstances {
		if instance.Endpoint != nil &&
			instance.Endpoint.Address != nil &&
			instance.Endpoint.Port != nil &&
			*instance.Endpoint.Address == host &&
			int(*instance.Endpoint.Port) == port {
			return &resp.DBInstances[i], nil
		}
	}
	return nil, fmt.Errorf("Failed to find RDS instance using endpoint-based search for host \"%s\" and port %d", host, port)
}

// FindRdsInstance finds and returns an RDS DBInstance for the given server config.
func FindRdsInstance(serverCfg config.ServerConfig, awsCfg aws.Config) (*rdstypes.DBInstance, error) {
	return findRdsInstanceWithContext(context.Background(), serverCfg, awsCfg)
}

func findRdsInstanceWithContext(ctx context.Context, serverCfg config.ServerConfig, awsCfg aws.Config) (*rdstypes.DBInstance, error) {
	client := NewRdsClient(awsCfg, serverCfg)

	if serverCfg.AwsDbInstanceID != "" {
		return findRdsInstanceByIdentifier(ctx, serverCfg.AwsDbInstanceID, client)
	}

	if serverCfg.AwsDbClusterID != "" {
		cluster, err := findRdsClusterByIdentifier(ctx, serverCfg.AwsDbClusterID, client)
		if err != nil {
			return nil, err
		}

		instanceID := ""
		for _, member := range cluster.DBClusterMembers {
			if member.IsClusterWriter == nil || member.DBInstanceIdentifier == nil {
				continue
			}
			isWriter := *member.IsClusterWriter
			if (serverCfg.AwsDbClusterReadonly && isWriter) ||
				(!serverCfg.AwsDbClusterReadonly && !isWriter) {
				continue
			}
			if instanceID == "" {
				instanceID = *member.DBInstanceIdentifier
			} else if serverCfg.AwsDbClusterReadonly {
				return nil, fmt.Errorf("Found more than one reader to monitor for read-only cluster \"%s\" (HINT: use specific instance IDs instead)", serverCfg.AwsDbClusterID)
			} else {
				return nil, fmt.Errorf("Unexpected multiple writers for cluster \"%s\"", serverCfg.AwsDbClusterID)
			}
		}
		if instanceID == "" {
			return nil, fmt.Errorf("Could not locate usable instance ID for cluster \"%s\" (readonly = %t)", serverCfg.AwsDbClusterID, serverCfg.AwsDbClusterReadonly)
		}
		return findRdsInstanceByIdentifier(ctx, instanceID, client)
	}

	// If neither instance ID nor cluster ID were specified, but we still have
	// an RDS system type, attempt to find the instance based on the hostname
	// (this is a long shot, but there are some cases where this helps)
	return findRdsInstanceByHostAndPort(ctx, serverCfg.GetDbHost(), serverCfg.GetDbPortOrDefault(), client)
}

// GetRdsParameter looks up a single named parameter from an RDS parameter group.
func GetRdsParameter(group *rdstypes.DBParameterGroupStatus, name string, client *rds.Client) (*rdstypes.Parameter, error) {
	return getRdsParameterWithContext(context.Background(), group, name, client)
}

func getRdsParameterWithContext(ctx context.Context, group *rdstypes.DBParameterGroupStatus, name string, client *rds.Client) (*rdstypes.Parameter, error) {
	params := &rds.DescribeDBParametersInput{
		DBParameterGroupName: group.DBParameterGroupName,
	}
	for {
		resp, err := client.DescribeDBParameters(ctx, params)
		if err != nil {
			return nil, err
		}
		for i, p := range resp.Parameters {
			if p.ParameterName != nil && *p.ParameterName == name {
				return &resp.Parameters[i], nil
			}
		}
		params.Marker = resp.Marker
		if params.Marker == nil {
			break
		}
	}
	return nil, nil
}

// RdsCloudWatchReader fetches CloudWatch metrics for an RDS instance/cluster.
type RdsCloudWatchReader struct {
	svc      *cloudwatch.Client
	instance string
	cluster  string
	logger   *util.Logger
}

// NewRdsCloudWatchReader creates an RdsCloudWatchReader backed by a CloudWatch client.
func NewRdsCloudWatchReader(awsCfg aws.Config, serverCfg config.ServerConfig, logger *util.Logger, instance string, cluster string) RdsCloudWatchReader {
	return RdsCloudWatchReader{
		svc:      NewCloudWatchClient(awsCfg, serverCfg),
		instance: instance,
		cluster:  cluster,
		logger:   logger,
	}
}

// GetRdsIntMetric gets an integer value from CloudWatch for the instance dimension.
func (reader RdsCloudWatchReader) GetRdsIntMetric(metricName string, unit string) int64 {
	return int64(reader.GetRdsFloatMetric(metricName, unit))
}

// GetRdsFloatMetric gets a float value from CloudWatch for the instance dimension.
func (reader RdsCloudWatchReader) GetRdsFloatMetric(metricName string, unit string) float64 {
	return reader.getMetric(metricName, unit, "DBInstanceIdentifier", reader.instance)
}

// GetRdsClusterIntMetric gets an integer value from CloudWatch using the cluster dimension.
// Uses a 3-hour lookback window since Aurora volume metrics like VolumeBytesUsed are
// reported infrequently (not continuously). Returns 0 if no datapoints are available.
func (reader RdsCloudWatchReader) GetRdsClusterIntMetric(metricName string, unit string) int64 {
	ctx := context.Background()
	resp, err := reader.svc.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String(metricName),
		Namespace:  aws.String("AWS/RDS"),
		Period:     aws.Int32(300),
		StartTime:  aws.Time(time.Now().Add(-3 * time.Hour)),
		Unit:       cwtypes.StandardUnit(unit),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
		Dimensions: []cwtypes.Dimension{{
			Name:  aws.String("DBClusterIdentifier"),
			Value: aws.String(reader.cluster),
		}},
	})
	if err != nil || len(resp.Datapoints) == 0 {
		return 0
	}

	var latest *cwtypes.Datapoint
	for i := range resp.Datapoints {
		dp := &resp.Datapoints[i]
		if latest == nil || (dp.Timestamp != nil && latest.Timestamp != nil && dp.Timestamp.After(*latest.Timestamp)) {
			latest = dp
		}
	}
	if latest != nil && latest.Average != nil {
		return int64(*latest.Average)
	}
	return 0
}

func (reader RdsCloudWatchReader) getMetric(metricName string, unit string, dimensionName string, dimensionValue string) float64 {
	ctx := context.Background()
	resp, err := reader.svc.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String(metricName),
		Namespace:  aws.String("AWS/RDS"),
		Period:     aws.Int32(60),
		StartTime:  aws.Time(time.Now().Add(-10 * time.Minute)),
		Unit:       cwtypes.StandardUnit(unit),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
		Dimensions: []cwtypes.Dimension{{
			Name:  aws.String(dimensionName),
			Value: aws.String(dimensionValue),
		}},
	})
	if err != nil {
		reader.logger.PrintVerbose(err.Error())
		return 0.0
	}
	if len(resp.Datapoints) == 0 || resp.Datapoints[0].Average == nil {
		return 0.0
	}
	return *resp.Datapoints[0].Average
}
