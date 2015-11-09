package systemstats

import (
    "fmt"
    "time"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    //"github.com/aws/aws-sdk-go/aws/request"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cloudwatch"
    //"github.com/aws/aws-sdk-go/service/rds"
)

func GetFromAws(awsAccessKeyId string, awsSecretAccessKey string) (system SnapshotSystem) {
  creds := credentials.NewStaticCredentials(awsAccessKeyId, awsSecretAccessKey, "")

  sess := session.New(&aws.Config{Credentials: creds, Region: aws.String("us-east-1")})
  //sess.Handlers.Send.PushFront(func(r *request.Request) {
    // Log every request made and its payload
  //  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
  //})

  //svc := rds.New(sess)

  // TODO: Need to match db host to correct RDS instance

  //params := &rds.DescribeDBInstancesInput{
  	//DBInstanceIdentifier: aws.String("String"),
  	// Filters: []*rds.Filter{
  	// 	{ // Required
  	// 		Name: aws.String("String"), // Required
  	// 		Values: []*string{ // Required
  	// 			aws.String("String"), // Required
  	// 			// More values...
  	// 		},
  	// 	},
  	// 	// More values...
  	// },
  	//Marker:     aws.String("String"),
  	//MaxRecords: aws.Int64(1),
  //}
  //resp, err := svc.DescribeDBInstances(params)

  //if err != nil {
  	// Print the error, cast err to awserr.Error to get the Code and
  	// Message from an error.
  	//fmt.Println(err.Error())
  	//return
  //}

  // Pretty-print the response data.
  //fmt.Println(resp)

  // CPUUtilization
  // DatabaseConnections
  // DiskQueueDepth
  // FreeStorageSpace
  // FreeableMemory
  // NetworkReceiveThroughput
  // NetworkTransmitThroughput
  // ReadLatency
  // ReadThroughput
  // SwapUsage
  // TransactionLogsDiskUsage
  // WriteLatency
  // WriteThroughput

  system.Storage.Perfdata.RdIos = GetMetric("pganalyze-production", "ReadIOPS", sess)
  system.Storage.Perfdata.WrIos = GetMetric("pganalyze-production", "WriteIOPS", sess)

  return
}

func GetMetric(instance string, metricName string, sess *session.Session) float64 {
  svc := cloudwatch.New(sess)

	params := &cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String(metricName),
		Namespace:  aws.String("AWS/RDS"),
		Period:     aws.Int64(60),
		StartTime:  aws.Time(time.Now().Add(-1 * time.Minute)),
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
		return 0
	}

	return *resp.Datapoints[0].Average
}
