package awsutil

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	collectorConfig "github.com/pganalyze/collector/config"
)

// GetAwsConfig returns an AWS config for the specified server configuration.
func GetAwsConfig(cfg collectorConfig.ServerConfig) (aws.Config, error) {
	return getAwsConfigWithContext(context.Background(), cfg)
}

func getAwsConfigWithContext(ctx context.Context, cfg collectorConfig.ServerConfig) (aws.Config, error) {
	var loadOpts []func(*awsconfig.LoadOptions) error

	loadOpts = append(loadOpts, awsconfig.WithRegion(cfg.AwsRegion))

	if cfg.HTTPClient != nil {
		loadOpts = append(loadOpts, awsconfig.WithHTTPClient(cfg.HTTPClient))
	}

	// Static credentials take precedence when configured; otherwise the default
	// chain (env vars, shared credentials file, EC2 IMDS) is used automatically.
	if cfg.AwsAccessKeyID != "" {
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

// NewRdsClient creates an RDS client with any custom endpoint from serverCfg applied.
func NewRdsClient(awsCfg aws.Config, serverCfg collectorConfig.ServerConfig) *rds.Client {
	var opts []func(*rds.Options)
	if serverCfg.AwsEndpointRdsURL != "" {
		url := serverCfg.AwsEndpointRdsURL
		opts = append(opts, func(o *rds.Options) { o.BaseEndpoint = &url })
	}
	if serverCfg.AwsEndpointSigningRegion != "" {
		region := serverCfg.AwsEndpointSigningRegion
		opts = append(opts, func(o *rds.Options) { o.Region = region })
	}
	return rds.NewFromConfig(awsCfg, opts...)
}

// NewCloudWatchClient creates a CloudWatch client with any custom endpoint from serverCfg applied.
func NewCloudWatchClient(awsCfg aws.Config, serverCfg collectorConfig.ServerConfig) *cloudwatch.Client {
	var opts []func(*cloudwatch.Options)
	if serverCfg.AwsEndpointCloudwatchURL != "" {
		url := serverCfg.AwsEndpointCloudwatchURL
		opts = append(opts, func(o *cloudwatch.Options) { o.BaseEndpoint = &url })
	}
	if serverCfg.AwsEndpointSigningRegion != "" {
		region := serverCfg.AwsEndpointSigningRegion
		opts = append(opts, func(o *cloudwatch.Options) { o.Region = region })
	}
	return cloudwatch.NewFromConfig(awsCfg, opts...)
}

// NewCloudWatchLogsClient creates a CloudWatch Logs client with any custom endpoint from serverCfg applied.
func NewCloudWatchLogsClient(awsCfg aws.Config, serverCfg collectorConfig.ServerConfig) *cloudwatchlogs.Client {
	var opts []func(*cloudwatchlogs.Options)
	if serverCfg.AwsEndpointCloudwatchLogsURL != "" {
		url := serverCfg.AwsEndpointCloudwatchLogsURL
		opts = append(opts, func(o *cloudwatchlogs.Options) { o.BaseEndpoint = &url })
	}
	if serverCfg.AwsEndpointSigningRegion != "" {
		region := serverCfg.AwsEndpointSigningRegion
		opts = append(opts, func(o *cloudwatchlogs.Options) { o.Region = region })
	}
	return cloudwatchlogs.NewFromConfig(awsCfg, opts...)
}
