package awsutil

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pganalyze/collector/config"
)

func GetAwsSession(config config.ServerConfig) (*session.Session, error) {
	var creds *credentials.Credentials

	if config.AwsAccessKeyID != "" {
		creds = credentials.NewStaticCredentials(config.AwsAccessKeyID, config.AwsSecretAccessKey, "")
	}

	return session.NewSession(&aws.Config{
		Credentials:                   creds,
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(config.AwsRegion),
		HTTPClient:                    config.HTTPClient,
	})
}

//sess.Handlers.Send.PushFront(func(r *request.Request) {
// Log every request made and its payload
//  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
//})
