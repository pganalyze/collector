package awsutil

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pganalyze/collector/config"
)

// GetAwsSession - Returns an AWS session for the specified server configuration
func GetAwsSession(config config.ServerConfig) (*session.Session, error) {
	var creds *credentials.Credentials

	if config.AwsAccessKeyID != "" {
		creds = credentials.NewStaticCredentials(config.AwsAccessKeyID, config.AwsSecretAccessKey, "")
	}

	customResolver := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		if service == endpoints.RdsServiceID && config.AwsEndpointRdsURL != "" {
			return endpoints.ResolvedEndpoint{
				URL:           config.AwsEndpointRdsURL,
				SigningRegion: config.AwsEndpointSigningRegion,
			}, nil
		}
		if service == endpoints.Ec2ServiceID && config.AwsEndpointEc2URL != "" {
			return endpoints.ResolvedEndpoint{
				URL:           config.AwsEndpointEc2URL,
				SigningRegion: config.AwsEndpointSigningRegion,
			}, nil
		}
		if service == endpoints.MonitoringServiceID && config.AwsEndpointCloudwatchURL != "" {
			return endpoints.ResolvedEndpoint{
				URL:           config.AwsEndpointCloudwatchURL,
				SigningRegion: config.AwsEndpointSigningRegion,
			}, nil
		}
		if service == endpoints.LogsServiceID && config.AwsEndpointCloudwatchLogsURL != "" {
			return endpoints.ResolvedEndpoint{
				URL:           config.AwsEndpointCloudwatchLogsURL,
				SigningRegion: config.AwsEndpointSigningRegion,
			}, nil
		}

		return endpoints.DefaultResolver().EndpointFor(service, region, optFns...)
	}

	if config.AwsAssumeRole != "" {
		sess, err := session.NewSession(&aws.Config{
			Credentials:                   creds,
			CredentialsChainVerboseErrors: aws.Bool(true),
			Region:                        aws.String(config.AwsRegion),
			HTTPClient:                    config.HTTPClient,
			EndpointResolver:              endpoints.ResolverFunc(customResolver),
		})
		if err != nil {
			return nil, err
		}
		creds = stscreds.NewCredentials(sess, config.AwsAssumeRole)
	}

	return session.NewSession(&aws.Config{
		Credentials:                   creds,
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(config.AwsRegion),
		HTTPClient:                    config.HTTPClient,
		EndpointResolver:              endpoints.ResolverFunc(customResolver),
	})
}

//sess.Handlers.Send.PushFront(func(r *request.Request) {
// Log every request made and its payload
//  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
//})
