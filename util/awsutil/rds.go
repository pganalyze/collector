package awsutil

import (
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

var IdentifierMap *util.TTLMap = util.NewTTLMap(5 * 60)
var ErrorCache *util.TTLMap = util.NewTTLMap(10 * 60)

func FindRdsIdentifier(config config.ServerConfig, sess *session.Session) (identifier string, err error) {
	identifier = IdentifierMap.Get(config.AwsDbInstanceID)
	if identifier != "" {
		return
	}

	cachedError := ErrorCache.Get(config.AwsDbInstanceID)
	if cachedError != "" {
		err = errors.New(cachedError)
		return
	}

	instance, err := FindRdsInstance(config, sess)
	if instance == nil {
		// Do nothing and return empty identifier (should we cache this?)
	} else if err == nil {
		identifier = *instance.DBInstanceIdentifier
		IdentifierMap.Put(config.AwsDbInstanceID, identifier)
	} else {
		ErrorCache.Put(config.AwsDbInstanceID, err.Error())
	}
	return
}

func FindRdsInstance(config config.ServerConfig, sess *session.Session) (instance *rds.DBInstance, err error) {
	var resp *rds.DescribeDBInstancesOutput
	var respCluster *rds.DescribeDBClustersOutput

	svc := rds.New(sess)

	var instanceID string
	if config.AwsDbInstanceID != "" {
		instanceID = config.AwsDbInstanceID
	}

	if instanceID == "" && config.AwsDbClusterID != "" {
		params := &rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(config.AwsDbClusterID),
		}

		respCluster, err = svc.DescribeDBClusters(params)
		if err != nil {
			return
		}

		if err == nil && len(respCluster.DBClusters) >= 1 {
			for _, clusterMember := range respCluster.DBClusters[0].DBClusterMembers {
				if *clusterMember.IsClusterWriter {
					instanceID = *clusterMember.DBInstanceIdentifier
				}
			}
		}
	}

	if instanceID != "" {
		params := &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(instanceID),
		}

		resp, err = svc.DescribeDBInstances(params)

		if err == nil && len(resp.DBInstances) >= 1 {
			instance = resp.DBInstances[0]
		}

		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "DBInstanceNotFound") {
				instance = nil
				err = nil
			}
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
