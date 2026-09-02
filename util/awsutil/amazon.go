package awsutil

import (
	"context"
	"net/http"

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

// getAwsConfig returns an AWS config for the given account key. It takes the
// AccountKey rather than a config.ServerConfig on purpose: the config is what
// determines which servers may share an Account, so every server config field
// used here must be part of the key. The HTTP client is passed separately
// (it is a constructed object, not comparable across servers), but the
// settings it is derived from are part of the key as well.
func getAwsConfig(ctx context.Context, key AccountKey, httpClient *http.Client) (aws.Config, error) {
	var loadOpts []func(*awsconfig.LoadOptions) error

	loadOpts = append(loadOpts, awsconfig.WithRegion(key.Region))

	// TODO: Global endpoint resolvers are deprecated and this should be migrated to service-specific
	// endpoint resolution, see https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-endpoints.html#migration
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == rds.ServiceID && key.EndpointRdsURL != "" {
			return aws.Endpoint{
				URL:           key.EndpointRdsURL,
				SigningRegion: key.EndpointSigningRegion,
			}, nil
		}
		if service == ec2.ServiceID && key.EndpointEc2URL != "" {
			return aws.Endpoint{
				URL:           key.EndpointEc2URL,
				SigningRegion: key.EndpointSigningRegion,
			}, nil
		}
		if service == cloudwatch.ServiceID && key.EndpointCloudwatchURL != "" {
			return aws.Endpoint{
				URL:           key.EndpointCloudwatchURL,
				SigningRegion: key.EndpointSigningRegion,
			}, nil
		}
		if service == cloudwatchlogs.ServiceID && key.EndpointCloudwatchLogsURL != "" {
			return aws.Endpoint{
				URL:           key.EndpointCloudwatchLogsURL,
				SigningRegion: key.EndpointSigningRegion,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	loadOpts = append(loadOpts, awsconfig.WithEndpointResolverWithOptions(customResolver))
	if httpClient != nil {
		loadOpts = append(loadOpts, awsconfig.WithHTTPClient(httpClient))
	}

	// Use a dedicated HTTP client with a short timeout for EC2 instance role
	// credential lookups, so the collector fails fast when not running on EC2
	// (instead of going through the general-purpose HTTP client above)
	loadOpts = append(loadOpts, awsconfig.WithEC2RoleCredentialOptions(func(o *ec2rolecreds.Options) {
		o.Client = imds.New(imds.Options{
			HTTPClient: config.CreateEC2IMDSHTTPClient(),
		})
	}))

	// Static credentials take precedence when fully configured; otherwise the
	// default chain (env vars, shared credentials file, EC2 IMDS) is used
	if key.AccessKeyID != "" && key.SecretAccessKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(key.AccessKeyID, key.SecretAccessKey, ""),
		))
	}

	baseCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return aws.Config{}, err
	}

	if key.WebIdentityTokenFile != "" && key.RoleArn != "" {
		stsClient := sts.NewFromConfig(baseCfg)
		webIdProvider := stscreds.NewWebIdentityRoleProvider(
			stsClient,
			key.RoleArn,
			stscreds.IdentityTokenFile(key.WebIdentityTokenFile),
		)
		baseCfg.Credentials = aws.NewCredentialsCache(webIdProvider)

		if key.AssumeRole != "" {
			stsClient2 := sts.NewFromConfig(baseCfg)
			assumeRoleProvider := stscreds.NewAssumeRoleProvider(stsClient2, key.AssumeRole)
			baseCfg.Credentials = aws.NewCredentialsCache(assumeRoleProvider)
		}
	} else if key.AssumeRole != "" {
		stsClient := sts.NewFromConfig(baseCfg)
		assumeRoleProvider := stscreds.NewAssumeRoleProvider(stsClient, key.AssumeRole)
		baseCfg.Credentials = aws.NewCredentialsCache(assumeRoleProvider)
	}

	return baseCfg, nil
}
