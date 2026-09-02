package awsutil

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/util"
)

// AccountKey identifies a distinct AWS API client configuration. Server
// configurations that agree on all these fields can safely share credentials
// and API clients (and their cached API results).
//
// Note that this does not include "Account ID" because we typically don't know
// that directly in the config, but instead it's inferred from the context
// (metadata service, credentials, assume role, etc).
type AccountKey struct {
	Region                    string
	AccessKeyID               string
	SecretAccessKey           string
	AssumeRole                string
	WebIdentityTokenFile      string
	RoleArn                   string
	EndpointSigningRegion     string
	EndpointRdsURL            string
	EndpointEc2URL            string
	EndpointCloudwatchURL     string
	EndpointCloudwatchLogsURL string

	// The HTTP client used for AWS API calls is derived from these settings,
	// so they are part of the sharing key as well
	APIBaseURL string
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

// Account holds the shared AWS API clients (and caches of API results) for
// all servers that connect to AWS the same way. Notably this ensures we only
// maintain one credentials cache (i.e. one sts:AssumeRole session) and
// benefit from cached RDS Describe* calls across all servers in an account.
type Account struct {
	Config         aws.Config
	RDS            *rds.Client
	CloudWatch     *cloudwatch.Client
	CloudWatchLogs *cloudwatchlogs.Client

	instanceList ttlCache[[]rdstypes.DBInstance]
	clusterList  ttlCache[[]rdstypes.DBCluster]

	// Fallback caches for single-object lookups, only used when the account-wide
	// listing is unavailable (e.g. denied by a scoped IAM policy)
	instances ttlCache[*rdstypes.DBInstance]
	clusters  ttlCache[*rdstypes.DBCluster]
}

var accountsMutex sync.Mutex
var accounts = make(map[AccountKey]*Account)

func accountKeyFromConfig(cfg config.ServerConfig) AccountKey {
	return AccountKey{
		Region:                    cfg.AwsRegion,
		AccessKeyID:               cfg.AwsAccessKeyID,
		SecretAccessKey:           cfg.AwsSecretAccessKey,
		AssumeRole:                cfg.AwsAssumeRole,
		WebIdentityTokenFile:      cfg.AwsWebIdentityTokenFile,
		RoleArn:                   cfg.AwsRoleArn,
		EndpointSigningRegion:     cfg.AwsEndpointSigningRegion,
		EndpointRdsURL:            cfg.AwsEndpointRdsURL,
		EndpointEc2URL:            cfg.AwsEndpointEc2URL,
		EndpointCloudwatchURL:     cfg.AwsEndpointCloudwatchURL,
		EndpointCloudwatchLogsURL: cfg.AwsEndpointCloudwatchLogsURL,
		APIBaseURL:                cfg.APIBaseURL,
		HTTPProxy:                 cfg.HTTPProxy,
		HTTPSProxy:                cfg.HTTPSProxy,
		NoProxy:                   cfg.NoProxy,
	}
}

// GetAccount returns the Account shared by all server configurations that
// connect to AWS the same way (see AccountKey), creating it on first use.
func GetAccount(ctx context.Context, cfg config.ServerConfig) (*Account, error) {
	key := accountKeyFromConfig(cfg)

	accountsMutex.Lock()
	defer accountsMutex.Unlock()

	if account, ok := accounts[key]; ok {
		return account, nil
	}

	awsCfg, err := getAwsConfig(ctx, key, cfg.HTTPClient)
	if err != nil {
		return nil, err
	}

	account := &Account{
		Config:         awsCfg,
		RDS:            rds.NewFromConfig(awsCfg),
		CloudWatch:     cloudwatch.NewFromConfig(awsCfg),
		CloudWatchLogs: cloudwatchlogs.NewFromConfig(awsCfg),
	}
	accounts[key] = account

	return account, nil
}

// ClearAccounts drops all shared account state, so the next GetAccount call
// creates fresh clients and caches. Called on configuration reload, since
// otherwise accounts of removed servers (or with outdated AWS settings)
// would be kept around indefinitely.
func ClearAccounts() {
	accountsMutex.Lock()
	defer accountsMutex.Unlock()
	accounts = make(map[AccountKey]*Account)
}

// GetRdsInstance looks up an RDS instance by identifier. This is served from
// the cached account-wide instance list when possible, so N monitored
// instances cost one DescribeDBInstances listing per TTL for the whole
// account (API rate limits are account-wide), rather than N describe calls.
//
// The result is shared with other callers and must not be modified.
func (a *Account) GetRdsInstance(ctx context.Context, instanceID string, logger *util.Logger) (*rdstypes.DBInstance, error) {
	instances, err := a.GetAllRdsInstances(ctx)
	if err == nil {
		for i := range instances {
			if instances[i].DBInstanceIdentifier != nil && *instances[i].DBInstanceIdentifier == instanceID {
				return &instances[i], nil
			}
		}
		return nil, fmt.Errorf("Could not find RDS instance \"%s\" among the %d instances visible in this AWS account and region (check aws_db_instance_id and aws_region)", instanceID, len(instances))
	}

	// When the listing itself is unavailable - e.g. denied by a scoped IAM policy
	// we fall back to a (cached) single-instance describe.
	logger.PrintVerbose("Could not list RDS instances (%s), falling back to individual lookup of instance \"%s\"", err, instanceID)
	return a.instances.get(ctx, instanceID, func(ctx context.Context) (*rdstypes.DBInstance, error) {
		resp, err := a.RDS.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(instanceID),
		})
		if err != nil {
			return nil, err
		}
		if len(resp.DBInstances) == 0 {
			return nil, fmt.Errorf("Unexpected empty result set for DescribeDBInstances with DBInstanceIdentifier = \"%s\"", instanceID)
		}
		return &resp.DBInstances[0], nil
	})
}

// GetRdsCluster looks up an RDS/Aurora cluster by identifier. Like
// GetRdsInstance this is served from the cached account-wide cluster list
// (which is authoritative when available), falling back to a (cached)
// single-cluster describe only when the listing is unavailable.
//
// The result is shared with other callers and must not be modified.
func (a *Account) GetRdsCluster(ctx context.Context, clusterID string, logger *util.Logger) (*rdstypes.DBCluster, error) {
	clusters, err := a.GetAllRdsClusters(ctx)
	if err == nil {
		for i := range clusters {
			if clusters[i].DBClusterIdentifier != nil && *clusters[i].DBClusterIdentifier == clusterID {
				return &clusters[i], nil
			}
		}
		return nil, fmt.Errorf("Could not find RDS cluster \"%s\" among the %d clusters visible in this AWS account and region (check aws_db_cluster_id and aws_region)", clusterID, len(clusters))
	}

	logger.PrintVerbose("Could not list RDS clusters (%s), falling back to individual lookup of cluster \"%s\"", err, clusterID)
	return a.clusters.get(ctx, clusterID, func(ctx context.Context) (*rdstypes.DBCluster, error) {
		resp, err := a.RDS.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(clusterID),
		})
		if err != nil {
			return nil, err
		}
		if len(resp.DBClusters) == 0 {
			return nil, fmt.Errorf("Unexpected empty result set for DescribeDBClusters with DBClusterIdentifier = \"%s\"", clusterID)
		}
		return &resp.DBClusters[0], nil
	})
}

// GetAllRdsInstances returns all RDS instances visible in the account and
// region, using cached results when available.
//
// The result is shared with other callers and must not be modified.
func (a *Account) GetAllRdsInstances(ctx context.Context) ([]rdstypes.DBInstance, error) {
	return a.instanceList.get(ctx, "all", func(ctx context.Context) ([]rdstypes.DBInstance, error) {
		var instances []rdstypes.DBInstance
		paginator := rds.NewDescribeDBInstancesPaginator(a.RDS, &rds.DescribeDBInstancesInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			instances = append(instances, page.DBInstances...)
		}
		return instances, nil
	})
}

// GetAllRdsClusters returns all RDS/Aurora clusters visible in the account
// and region, using cached results when available.
//
// The result is shared with other callers and must not be modified.
func (a *Account) GetAllRdsClusters(ctx context.Context) ([]rdstypes.DBCluster, error) {
	return a.clusterList.get(ctx, "all", func(ctx context.Context) ([]rdstypes.DBCluster, error) {
		var clusters []rdstypes.DBCluster
		paginator := rds.NewDescribeDBClustersPaginator(a.RDS, &rds.DescribeDBClustersInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			clusters = append(clusters, page.DBClusters...)
		}
		return clusters, nil
	})
}
