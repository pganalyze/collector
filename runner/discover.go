package runner

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/pganalyze/collector/config"
	"github.com/pganalyze/collector/state"
	"github.com/pganalyze/collector/util"
	"github.com/pganalyze/collector/util/awsutil"
)

// How often the desired server set of discovery-enabled config sections is
// re-determined. Newly provisioned servers (e.g. an added Aurora reader) are
// picked up within this interval plus the describe cache TTL.
const serverDiscoveryInterval = 1 * time.Minute

// Instance statuses for which monitoring is not possible (or about to become
// impossible), and which therefore exclude a discovered instance. Transient
// statuses like "rebooting" or "backing-up" intentionally keep the instance
// monitored to avoid removing and re-adding servers during maintenance.
var excludedInstanceStatuses = map[string]bool{
	"creating": true,
	"deleting": true,
	"failed":   true,
	"stopped":  true,
	"stopping": true,
}

// Last known successful discovery result per config section. Used as a
// fallback when AWS API calls fail, so a temporary API error never causes
// discovered servers to be dropped from monitoring (an instance only counts
// as removed based on a successful API response that no longer includes it).
var lastKnownDiscovery = struct {
	sync.Mutex
	bySection map[string][]config.ServerConfig
}{bySection: make(map[string][]config.ServerConfig)}

// ExpandServerConfigs resolves config sections that enable server discovery
// (e.g. aws_db_cluster_members = all) into concrete per-instance server
// configs. Sections without discovery pass through unchanged.
func ExpandServerConfigs(ctx context.Context, sectionConfigs []config.ServerConfig, logger *util.Logger) []config.ServerConfig {
	// Instances that are already statically configured take precedence over
	// discovering the same instance as part of a cluster
	staticInstances := make(map[string]bool)
	for _, cfg := range sectionConfigs {
		if !cfg.DiscoversServers() && cfg.AwsDbInstanceID != "" {
			staticInstances[cfg.AwsRegion+"/"+cfg.AwsDbInstanceID] = true
		}
	}

	var expanded []config.ServerConfig
	for _, cfg := range sectionConfigs {
		if !cfg.DiscoversServers() {
			expanded = append(expanded, cfg)
			continue
		}

		prefixedLogger := logger.WithPrefix(cfg.SectionName)
		memberConfigs, err := discoverClusterMemberConfigs(ctx, cfg, staticInstances, prefixedLogger)

		lastKnownDiscovery.Lock()
		if err != nil {
			// Note this may briefly reflect an outdated config after a reload
			// changed the section, until the next successful discovery run
			memberConfigs = lastKnownDiscovery.bySection[cfg.SectionName]
			prefixedLogger.PrintWarning("Could not discover servers for cluster \"%s\", keeping %d known server(s): %s", cfg.AwsDbClusterID, len(memberConfigs), err)
		} else {
			lastKnownDiscovery.bySection[cfg.SectionName] = memberConfigs
		}
		lastKnownDiscovery.Unlock()

		expanded = append(expanded, memberConfigs...)
	}
	return expanded
}

// Cluster statuses for which member discovery is skipped (the cluster has no
// usable instances yet, or won't have them much longer)
var excludedClusterStatuses = map[string]bool{
	"creating":  true,
	"deleting":  true,
	"preparing": true,
}

// discoverClusterMemberConfigs determines a server config for each member
// instance of the section's cluster, and - when enabled - of all clusters
// cloned from it (or that it was cloned from)
func discoverClusterMemberConfigs(ctx context.Context, sectionCfg config.ServerConfig, staticInstances map[string]bool, logger *util.Logger) ([]config.ServerConfig, error) {
	account, err := awsutil.GetAccount(ctx, sectionCfg)
	if err != nil {
		return nil, err
	}

	cluster, err := account.GetRdsCluster(ctx, sectionCfg.AwsDbClusterID, logger)
	if err != nil {
		return nil, err
	}

	memberConfigs, err := clusterMemberConfigs(ctx, account, sectionCfg, cluster, staticInstances, true, logger)
	if err != nil {
		return nil, err
	}

	// Aurora clones are separate clusters that share a clone group with the
	// cluster they were created from (the clone group only exists once the
	// first clone has been created)
	cloneGroupID := util.StringPtrToString(cluster.CloneGroupId)
	if sectionCfg.AwsDbClusterClones && cloneGroupID != "" {
		// Note this requires the account-wide cluster listing (unlike member
		// discovery, which can fall back to individual lookups when a scoped
		// IAM policy denies the listing)
		clusters, err := account.GetAllRdsClusters(ctx)
		if err != nil {
			return nil, err
		}
		for i, cloneCluster := range clusters {
			if util.StringPtrToString(cloneCluster.CloneGroupId) != cloneGroupID {
				continue
			}
			cloneClusterID := util.StringPtrToString(cloneCluster.DBClusterIdentifier)
			if cloneClusterID == sectionCfg.AwsDbClusterID {
				continue
			}
			status := util.StringPtrToString(cloneCluster.Status)
			if excludedClusterStatuses[status] {
				logger.PrintVerbose("Skipping clone cluster \"%s\" with status \"%s\"", cloneClusterID, status)
				continue
			}
			cloneMemberConfigs, err := clusterMemberConfigs(ctx, account, sectionCfg, &clusters[i], staticInstances, false, logger)
			if err != nil {
				return nil, err
			}
			memberConfigs = append(memberConfigs, cloneMemberConfigs...)
		}
	}

	return memberConfigs, nil
}

// clusterMemberConfigs determines the server configs for all usable member
// instances of one cluster. The section's fallback identity is only assigned
// when monitoring the writer of the section's own cluster (not a clone).
func clusterMemberConfigs(ctx context.Context, account *awsutil.Account, sectionCfg config.ServerConfig, cluster *rdstypes.DBCluster, staticInstances map[string]bool, sectionIdentityFallback bool, logger *util.Logger) ([]config.ServerConfig, error) {
	clusterID := util.StringPtrToString(cluster.DBClusterIdentifier)

	var memberConfigs []config.ServerConfig
	var lastErr error
	for _, member := range cluster.DBClusterMembers {
		instanceID := util.StringPtrToString(member.DBInstanceIdentifier)
		if instanceID == "" {
			continue
		}
		if staticInstances[sectionCfg.AwsRegion+"/"+instanceID] {
			logger.PrintVerbose("Skipping cluster member \"%s\", since it is already configured explicitly", instanceID)
			continue
		}

		// This is typically served from the cached account-wide instance list
		instance, err := account.GetRdsInstance(ctx, instanceID, logger)
		if err != nil {
			// Skip just this member (it may have been deleted moments ago), but
			// track the error in case all lookups fail (e.g. denied by IAM)
			logger.PrintVerbose("Skipping cluster member \"%s\": %s", instanceID, err)
			lastErr = err
			continue
		}
		status := util.StringPtrToString(instance.DBInstanceStatus)
		if excludedInstanceStatuses[status] {
			logger.PrintVerbose("Skipping cluster member \"%s\" with status \"%s\"", instanceID, status)
			continue
		}
		if instance.Endpoint == nil || instance.Endpoint.Address == nil || instance.Endpoint.Port == nil {
			logger.PrintVerbose("Skipping cluster member \"%s\", since it has no endpoint (yet)", instanceID)
			continue
		}

		memberConfigs = append(memberConfigs, memberServerConfig(sectionCfg, clusterID, instanceID, *instance.Endpoint.Address, int(*instance.Endpoint.Port), sectionIdentityFallback && util.BoolPtrToBool(member.IsClusterWriter)))
	}

	// If every member lookup failed we treat this as a discovery error (keeping
	// the last known set), since it likely indicates an API or IAM policy issue
	// rather than the members being gone
	if len(memberConfigs) == 0 && lastErr != nil {
		return nil, lastErr
	}

	// Note that zero members is a legitimate result (e.g. for a stopped
	// cluster), and distinct from an error (which keeps the last known set)
	if len(memberConfigs) == 0 {
		logger.PrintVerbose("Found no usable member instances for cluster \"%s\"", clusterID)
	}

	return memberConfigs, nil
}

// memberServerConfig derives the server config for a single discovered cluster
// member instance from its section's config. sectionIdentityFallback is set
// for the writer of the section's own cluster, which takes over the identity
// previously used by the cluster-level config as its fallback identity.
func memberServerConfig(sectionCfg config.ServerConfig, clusterID string, instanceID string, endpointHost string, endpointPort int, sectionIdentityFallback bool) config.ServerConfig {
	cfg := sectionCfg

	cfg.SectionName = sectionCfg.SectionName + "/" + instanceID
	cfg.AwsDbClusterID = clusterID
	cfg.AwsDbClusterReadonly = false
	cfg.AwsDbInstanceID = instanceID

	// If the section connects through a db_url (pointing at a cluster endpoint),
	// resolve it into its parts so the instance endpoint can be used instead.
	// Explicitly configured settings take precedence, like in GetPqOpenString.
	if cfg.DbURL != "" {
		if u, err := url.Parse(cfg.DbURL); err == nil {
			if u.User != nil {
				if cfg.DbUsername == "" {
					cfg.DbUsername = u.User.Username()
				}
				if cfg.DbPassword == "" {
					cfg.DbPassword, _ = u.User.Password()
				}
			}
			if cfg.DbName == "" && len(u.Path) > 0 {
				cfg.DbName = u.Path[1:]
			}
			for _, querySplit := range strings.Split(u.RawQuery, "&") {
				keyValue := strings.SplitN(querySplit, "=", 2)
				if len(keyValue) != 2 {
					continue
				}
				switch keyValue[0] {
				case "sslmode":
					if cfg.DbSslMode == "" {
						cfg.DbSslMode = keyValue[1]
					}
				case "sslrootcert":
					if cfg.DbSslRootCert == "" {
						cfg.DbSslRootCert = keyValue[1]
					}
				case "sslcert":
					if cfg.DbSslCert == "" {
						cfg.DbSslCert = keyValue[1]
					}
				case "sslkey":
					if cfg.DbSslKey == "" {
						cfg.DbSslKey = keyValue[1]
					}
				}
			}
		}
		cfg.DbURL = ""
	}
	cfg.DbHost = endpointHost
	cfg.DbPort = endpointPort

	// Each member gets its own per-instance identity in pganalyze
	cfg.SystemType = "amazon_rds"
	cfg.SystemID = instanceID
	if cfg.AwsAccountID != "" {
		cfg.SystemScope = cfg.AwsRegion + "/" + cfg.AwsAccountID
	} else {
		cfg.SystemScope = cfg.AwsRegion
	}

	// The section's identity is what a cluster-level config reported as before
	// members were discovered individually - point the writer's fallback
	// identity at it, so pganalyze can match up the existing server
	if sectionIdentityFallback && !sectionCfg.AwsDbClusterReadonly {
		cfg.SystemIDFallback = sectionCfg.SystemID
		cfg.SystemTypeFallback = sectionCfg.SystemType
		cfg.SystemScopeFallback = sectionCfg.SystemScope
	} else {
		cfg.SystemIDFallback = ""
		cfg.SystemTypeFallback = ""
		cfg.SystemScopeFallback = ""
	}

	cfg.Identifier = config.ServerIdentifier{
		APIKey:      cfg.APIKey,
		APIBaseURL:  cfg.APIBaseURL,
		SystemID:    cfg.SystemID,
		SystemType:  cfg.SystemType,
		SystemScope: cfg.SystemScope,
	}

	return cfg
}

// SetupServerDiscovery periodically re-runs server discovery and updates the
// monitored server list with any added or removed servers
func SetupServerDiscovery(ctx context.Context, wg *sync.WaitGroup, serverList *state.ServerList, sectionConfigs []config.ServerConfig, opts state.CollectionOpts, logger *util.Logger) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(serverDiscoveryInterval)
		defer ticker.Stop()

		missCounts := make(map[config.ServerIdentifier]int)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				desiredConfigs := ExpandServerConfigs(ctx, sectionConfigs, logger)
				desiredConfigs = applyRemovalHysteresis(desiredConfigs, serverList.Load(), missCounts)
				RefreshServers(ctx, serverList, desiredConfigs, opts, logger)
			}
		}
	}()
}

// applyRemovalHysteresis keeps currently monitored servers in the desired set
// until they have been absent from two consecutive discovery runs, so a single
// inconsistent API response does not remove a server
func applyRemovalHysteresis(desiredConfigs []config.ServerConfig, current []*state.Server, missCounts map[config.ServerIdentifier]int) []config.ServerConfig {
	desired := make(map[config.ServerIdentifier]bool, len(desiredConfigs))
	for _, cfg := range desiredConfigs {
		desired[cfg.Identifier] = true
		delete(missCounts, cfg.Identifier)
	}

	for _, server := range current {
		identifier := server.Config.Identifier
		if desired[identifier] {
			continue
		}
		missCounts[identifier]++
		if missCounts[identifier] < 2 {
			desiredConfigs = append(desiredConfigs, server.Config)
		} else {
			delete(missCounts, identifier)
		}
	}

	return desiredConfigs
}
