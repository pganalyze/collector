package util

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/lfittl/pganalyze-collector-next/config"
)

func FindRdsInstance(config config.Config, sess *session.Session) (instance *rds.DBInstance, err error) {
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

// GetRdsIntMetric - Gets an integer value from Cloudwatch
func GetRdsIntMetric(instance string, metricName string, unit string, sess *session.Session) *int64 {
	value := GetRdsFloatMetric(instance, metricName, unit, sess)
	if value == nil {
		return nil
	}
	var valueInt = int64(*value)
	return &valueInt
}

// GetRdsFloatMetric - Gets a float value from Cloudwatch
func GetRdsFloatMetric(instance string, metricName string, unit string, sess *session.Session) *float64 {
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
