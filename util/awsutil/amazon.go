package awsutil

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pganalyze/collector/config"
)

func GetAwsSession(config config.ServerConfig) *session.Session {
	var creds *credentials.Credentials

	if config.AwsAccessKeyID != "" {
		creds = credentials.NewStaticCredentials(config.AwsAccessKeyID, config.AwsSecretAccessKey, "")
	}

	return session.New(&aws.Config{Credentials: creds, Region: aws.String(config.AwsRegion)})
}

//sess.Handlers.Send.PushFront(func(r *request.Request) {
// Log every request made and its payload
//  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
//})
