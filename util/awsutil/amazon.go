package awsutil

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pganalyze/collector/config"
)

// GetAwsSession - Returns an AWS session for the specified server cfguration
func GetAwsSession(cfg config.ServerConfig) (*session.Session, error) {
	var providers []credentials.Provider

	customResolver := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		if service == endpoints.RdsServiceID && cfg.AwsEndpointRdsURL != "" {
			return endpoints.ResolvedEndpoint{
				URL:           cfg.AwsEndpointRdsURL,
				SigningRegion: cfg.AwsEndpointSigningRegion,
			}, nil
		}
		if service == endpoints.Ec2ServiceID && cfg.AwsEndpointEc2URL != "" {
			return endpoints.ResolvedEndpoint{
				URL:           cfg.AwsEndpointEc2URL,
				SigningRegion: cfg.AwsEndpointSigningRegion,
			}, nil
		}
		if service == endpoints.MonitoringServiceID && cfg.AwsEndpointCloudwatchURL != "" {
			return endpoints.ResolvedEndpoint{
				URL:           cfg.AwsEndpointCloudwatchURL,
				SigningRegion: cfg.AwsEndpointSigningRegion,
			}, nil
		}
		if service == endpoints.LogsServiceID && cfg.AwsEndpointCloudwatchLogsURL != "" {
			return endpoints.ResolvedEndpoint{
				URL:           cfg.AwsEndpointCloudwatchLogsURL,
				SigningRegion: cfg.AwsEndpointSigningRegion,
			}, nil
		}

		return endpoints.DefaultResolver().EndpointFor(service, region, optFns...)
	}

	if cfg.AwsAccessKeyID != "" {
		providers = append(providers, &credentials.StaticProvider{
			Value: credentials.Value{
				AccessKeyID:     cfg.AwsAccessKeyID,
				SecretAccessKey: cfg.AwsSecretAccessKey,
				SessionToken:    "",
			},
		})
	}

	// add default providers
	providers = append(providers, &credentials.EnvProvider{})
	providers = append(providers, &credentials.SharedCredentialsProvider{Filename: "", Profile: ""})

	// add the metadata service
	def := defaults.Get()
	def.Config.HTTPClient = config.CreateEC2IMDSHTTPClient(cfg)
	def.Config.MaxRetries = aws.Int(2)
	providers = append(providers, defaults.RemoteCredProvider(*def.Config, def.Handlers))

	creds := credentials.NewChainCredentials(providers)

	if cfg.AwsAssumeRole != "" || (cfg.AwsWebIdentityTokenFile != "" && cfg.AwsRoleArn != "") {
		sess, err := session.NewSession(&aws.Config{
			Credentials:                   creds,
			CredentialsChainVerboseErrors: aws.Bool(true),
			Region:                        aws.String(cfg.AwsRegion),
			HTTPClient:                    cfg.HTTPClient,
			EndpointResolver:              endpoints.ResolverFunc(customResolver),
		})
		if err != nil {
			return nil, err
		}
		if cfg.AwsAssumeRole != "" {
			creds = stscreds.NewCredentials(sess, cfg.AwsAssumeRole)
		} else if cfg.AwsWebIdentityTokenFile != "" && cfg.AwsRoleArn != "" {
			creds = stscreds.NewWebIdentityCredentials(sess, cfg.AwsRoleArn, "", cfg.AwsWebIdentityTokenFile)
		}
	}

	return session.NewSession(&aws.Config{
		Credentials:                   creds,
		CredentialsChainVerboseErrors: aws.Bool(true),
		Region:                        aws.String(cfg.AwsRegion),
		HTTPClient:                    cfg.HTTPClient,
		EndpointResolver:              endpoints.ResolverFunc(customResolver),
	})
}

//sess.Handlers.Send.PushFront(func(r *request.Request) {
// Log every request made and its payload
//  fmt.Printf("Request: %s/%s, Payload: %s\n", r.ClientInfo.ServiceName, r.Operation, r.Params)
//})
