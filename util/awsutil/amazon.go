package awsutil

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pganalyze/collector/config"
)

// GetAwsConfig returns an AWS config for the specified server configuration.
func GetAwsConfig(ctx context.Context, cfg config.ServerConfig) (aws.Config, error) {
	var loadOpts []func(*awsconfig.LoadOptions) error

	loadOpts = append(loadOpts, awsconfig.WithRegion(cfg.AwsRegion))

	// TODO: Global endpoint resolvers are deprecated and this should be migrated to service-specific
	// endpoint resolution, see https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-endpoints.html#migration
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == rds.ServiceID && cfg.AwsEndpointRdsURL != "" {
			return aws.Endpoint{
				URL:           cfg.AwsEndpointRdsURL,
				SigningRegion: cfg.AwsEndpointSigningRegion,
			}, nil
		}
		if service == ec2.ServiceID && cfg.AwsEndpointEc2URL != "" {
			return aws.Endpoint{
				URL:           cfg.AwsEndpointEc2URL,
				SigningRegion: cfg.AwsEndpointSigningRegion,
			}, nil
		}
		if service == cloudwatch.ServiceID && cfg.AwsEndpointCloudwatchURL != "" {
			return aws.Endpoint{
				URL:           cfg.AwsEndpointCloudwatchURL,
				SigningRegion: cfg.AwsEndpointSigningRegion,
			}, nil
		}
		if service == cloudwatchlogs.ServiceID && cfg.AwsEndpointCloudwatchLogsURL != "" {
			return aws.Endpoint{
				URL:           cfg.AwsEndpointCloudwatchLogsURL,
				SigningRegion: cfg.AwsEndpointSigningRegion,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	loadOpts = append(loadOpts, awsconfig.WithEndpointResolverWithOptions(customResolver))
	if cfg.HTTPClient != nil {
		loadOpts = append(loadOpts, awsconfig.WithHTTPClient(cfg.HTTPClient))
	}

	// Use a dedicated HTTP client with a short timeout for EC2 instance role
	// credential lookups, so the collector fails fast when not running on EC2
	// (instead of going through the general-purpose HTTP client above)
	loadOpts = append(loadOpts, awsconfig.WithEC2RoleCredentialOptions(func(o *ec2rolecreds.Options) {
		o.Client = imds.New(imds.Options{
			HTTPClient: config.CreateEC2IMDSHTTPClient(cfg),
		})
	}))

	// Static credentials take precedence when fully configured; otherwise the
	// default chain (env vars, shared credentials file, EC2 IMDS) is used
	if cfg.AwsAccessKeyID != "" && cfg.AwsSecretAccessKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AwsAccessKeyID, cfg.AwsSecretAccessKey, ""),
		))
	}

	baseCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return aws.Config{}, err
	}

	if cfg.AwsWebIdentityTokenFile != "" && cfg.AwsRoleArn != "" {
		stsClient := sts.NewFromConfig(baseCfg)
		webIdProvider := stscreds.NewWebIdentityRoleProvider(
			stsClient,
			cfg.AwsRoleArn,
			stscreds.IdentityTokenFile(cfg.AwsWebIdentityTokenFile),
		)
		baseCfg.Credentials = aws.NewCredentialsCache(webIdProvider)

		if cfg.AwsAssumeRole != "" {
			stsClient2 := sts.NewFromConfig(baseCfg)
			assumeRoleProvider := stscreds.NewAssumeRoleProvider(stsClient2, cfg.AwsAssumeRole)
			baseCfg.Credentials = aws.NewCredentialsCache(assumeRoleProvider)
		}
	} else if cfg.AwsAssumeRole != "" {
		stsClient := sts.NewFromConfig(baseCfg)
		assumeRoleProvider := stscreds.NewAssumeRoleProvider(stsClient, cfg.AwsAssumeRole)
		baseCfg.Credentials = aws.NewCredentialsCache(assumeRoleProvider)
	}

	return baseCfg, nil
}

// NewRdsClient creates an RDS client
func NewRdsClient(awsCfg aws.Config, serverCfg config.ServerConfig) *rds.Client {
	return rds.NewFromConfig(awsCfg)
}

// NewCloudWatchClient creates a CloudWatch client
func NewCloudWatchClient(awsCfg aws.Config, serverCfg config.ServerConfig) *cloudwatch.Client {
	return cloudwatch.NewFromConfig(awsCfg)
}

// NewCloudWatchLogsClient creates a CloudWatch Logs client
func NewCloudWatchLogsClient(awsCfg aws.Config, serverCfg config.ServerConfig) *cloudwatchlogs.Client {
	return cloudwatchlogs.NewFromConfig(awsCfg)
}
