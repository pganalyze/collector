package util

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pganalyze/collector/config"
)

func GetAwsSession(config config.DatabaseConfig) *session.Session {
	var creds *credentials.Credentials

	if config.AwsAccessKeyID != "" {
		creds = credentials.NewStaticCredentials(config.AwsAccessKeyID, config.AwsSecretAccessKey, "")
	} else {
		creds = credentials.NewCredentials(&ec2rolecreds.EC2RoleProvider{
			Client: ec2metadata.New(session.New()),
		})
	}

	return session.New(&aws.Config{Credentials: creds, Region: aws.String(config.AwsRegion)})
}

//sess.Handlers.Send.PushFront(func(r *request.Request) {
// Log every request made and its payload
//  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
//})
