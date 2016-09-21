package awsutil

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

func FindRdsInstance(config config.ServerConfig, sess *session.Session) (instance *rds.DBInstance, err error) {
	var resp *rds.DescribeDBInstancesOutput

	svc := rds.New(sess)

	if config.AwsDbInstanceID != "" {
		params := &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(config.AwsDbInstanceID),
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
