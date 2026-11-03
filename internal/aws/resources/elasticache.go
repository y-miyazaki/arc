package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// ElastiCacheCollector collects ElastiCache resources.
// It uses dependency injection to manage ElastiCache clients for multiple regions.
type ElastiCacheCollector struct {
	clients      map[string]*elasticache.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewElastiCacheCollector creates a new ElastiCache collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create ElastiCache clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *ElastiCacheCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewElastiCacheCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*ElastiCacheCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *elasticache.Client {
		return elasticache.NewFromConfig(*c, func(o *elasticache.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ElastiCache clients: %w", err)
	}

	return &ElastiCacheCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Collect collects ElastiCache resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *ElastiCacheCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Get all security groups to resolve names efficiently
	sgMap, err := c.nameResolver.GetAllSecurityGroups(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get security groups: %w", err)
	}

	// Describe Replication Groups
	rgPaginator := elasticache.NewDescribeReplicationGroupsPaginator(svc, &elasticache.DescribeReplicationGroupsInput{})

	for rgPaginator.HasMorePages() {
		page, pageErr := rgPaginator.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to describe replication groups: %w", pageErr)
		}
		for i := range page.ReplicationGroups {
			rg := &page.ReplicationGroups[i]

			// Determine Engine from first member or cache cluster if possible, but RG doesn't have Engine field directly in struct usually?
			// Actually DescribeReplicationGroups output has Engine field? AWS SDK v2 docs say ReplicationGroup has no Engine field directly,
			// but it might be inferred. Wait, the bash script extracts '.Engine'.
			// Checking SDK: ReplicationGroup struct DOES NOT have Engine field.
			// However, the bash script uses `aws elasticache describe-replication-groups` and extracts `.Engine`.
			// Let's check if it's available in the raw output or if we need to look at MemberClusters.
			// For now, we'll leave Engine empty for RG or try to get it from cache clusters later.
			// Actually, looking at the bash script, it seems to assume it's there.
			// Let's look at MemberClusters.

			// We will store RG data to process later or just add it now.
			// Since we need to link Cache Clusters to RG, we can process RGs first.

			resources = append(resources, NewResource(&ResourceInput{
				Category:     categoryElastiCache,
				SubCategory1: "ReplicationGroup",
				Name:         rg.ReplicationGroupId,
				Region:       region,
				ARN:          rg.ARN,
				RawData: map[string]any{
					"Description":              rg.Description,
					keyReplicationGroupID:      rg.ReplicationGroupId,
					keyNodeType:                rg.CacheNodeType,
					"NodeGroups":               helpers.FormatJSONIndentOrRaw(rg.NodeGroups),
					"MultiAZ":                  rg.MultiAZ,
					"AutomaticFailover":        rg.AutomaticFailover,
					keyAuthTokenEnabled:        rg.AuthTokenEnabled,
					keyEncryptedAtRest:         rg.AtRestEncryptionEnabled,
					keyEncryptedTransit:        rg.TransitEncryptionEnabled,
					keyAutoMinorVersionUpgrade: rg.AutoMinorVersionUpgrade,
					keySnapshotRetentionLimit:  rg.SnapshotRetentionLimit,
					keySnapshotWindow:          rg.SnapshotWindow,
					keyCreateTime:              rg.ReplicationGroupCreateTime,
					keyStatus:                  rg.Status,
				},
			}))
		}
	}

	// Track processed cluster IDs to avoid duplicates
	processedClusters := make(map[string]bool)

	// Process ReplicationGroup member clusters
	for rgPaginator2 := elasticache.NewDescribeReplicationGroupsPaginator(svc, &elasticache.DescribeReplicationGroupsInput{}); rgPaginator2.HasMorePages(); {
		page, pageErr := rgPaginator2.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to describe replication groups for clusters: %w", pageErr)
		}
		for i := range page.ReplicationGroups {
			rg := &page.ReplicationGroups[i]
			rgID := helpers.StringValue(rg.ReplicationGroupId)

			// Get cache clusters for this replication group
			ccPaginator := elasticache.NewDescribeCacheClustersPaginator(svc, &elasticache.DescribeCacheClustersInput{})
			for ccPaginator.HasMorePages() {
				ccPage, ccErr := ccPaginator.NextPage(ctx)
				if ccErr != nil {
					continue
				}

				for j := range ccPage.CacheClusters {
					cc := &ccPage.CacheClusters[j]
					ccID := helpers.StringValue(cc.CacheClusterId)
					ccRGID := helpers.StringValue(cc.ReplicationGroupId)

					// Only process clusters belonging to this replication group
					if ccRGID != rgID {
						continue
					}

					processedClusters[ccID] = true

					// Cache Parameter Group
					ccParamGroup := ""
					if cc.CacheParameterGroup != nil {
						ccParamGroup = helpers.StringValue(cc.CacheParameterGroup.CacheParameterGroupName)
					}

					// Security Groups
					var sgIDs []*string
					for k := range cc.SecurityGroups {
						sg := &cc.SecurityGroups[k]
						sgIDs = append(sgIDs, sg.SecurityGroupId)
					}

					resources = append(resources, NewResource(&ResourceInput{
						Category:     categoryElastiCache,
						SubCategory1: "",
						SubCategory2: "CacheCluster",
						Name:         cc.CacheClusterId,
						Region:       region,
						ARN:          cc.ARN,
						RawData: map[string]any{
							"ClusterID":                  cc.CacheClusterId,
							keyReplicationGroupID:        cc.ReplicationGroupId,
							"Engine":                     cc.Engine,
							"Version":                    cc.EngineVersion,
							keyNodeType:                  cc.CacheNodeType,
							"NumNodes":                   cc.NumCacheNodes,
							"CacheParameterGroup":        ccParamGroup,
							"SecurityGroup":              helpers.ResolveNamesFromMap(sgIDs, sgMap),
							keyAuthTokenEnabled:          cc.AuthTokenEnabled,
							keyEncryptedAtRest:           cc.AtRestEncryptionEnabled,
							keyEncryptedTransit:          cc.TransitEncryptionEnabled,
							keyAutoMinorVersionUpgrade:   cc.AutoMinorVersionUpgrade,
							"PreferredMaintenanceWindow": cc.PreferredMaintenanceWindow,
							keySnapshotRetentionLimit:    cc.SnapshotRetentionLimit,
							keySnapshotWindow:            cc.SnapshotWindow,
							keyCreateTime:                cc.CacheClusterCreateTime,
							keyStatus:                    cc.CacheClusterStatus,
						},
					}))
				}
			}
		}
	}

	// Describe Standalone Cache Clusters (not part of any replication group)
	ccPaginator := elasticache.NewDescribeCacheClustersPaginator(svc, &elasticache.DescribeCacheClustersInput{})
	for ccPaginator.HasMorePages() {
		page, pageErr := ccPaginator.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to describe cache clusters: %w", pageErr)
		}

		for i := range page.CacheClusters {
			cc := &page.CacheClusters[i]
			ccID := helpers.StringValue(cc.CacheClusterId)
			ccRGID := helpers.StringValue(cc.ReplicationGroupId)

			// Skip if already processed as part of a replication group or if it belongs to one
			if processedClusters[ccID] || ccRGID != "" {
				continue
			}

			// Cache Parameter Group
			ccParamGroup := ""
			if cc.CacheParameterGroup != nil {
				ccParamGroup = helpers.StringValue(cc.CacheParameterGroup.CacheParameterGroupName)
			}

			// Security Groups
			var sgIDs []*string
			for j := range cc.SecurityGroups {
				sg := &cc.SecurityGroups[j]
				sgIDs = append(sgIDs, sg.SecurityGroupId)
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:     categoryElastiCache,
				SubCategory1: "CacheCluster",
				Name:         cc.CacheClusterId,
				Region:       region,
				ARN:          cc.ARN,
				RawData: map[string]any{
					"ClusterID":                  cc.CacheClusterId,
					keyReplicationGroupID:        cc.ReplicationGroupId,
					"Engine":                     cc.Engine,
					"Version":                    cc.EngineVersion,
					keyNodeType:                  cc.CacheNodeType,
					"NumNodes":                   cc.NumCacheNodes,
					"CacheParameterGroup":        ccParamGroup,
					"SecurityGroup":              helpers.ResolveNamesFromMap(sgIDs, sgMap),
					keyAuthTokenEnabled:          cc.AuthTokenEnabled,
					keyEncryptedAtRest:           cc.AtRestEncryptionEnabled,
					keyEncryptedTransit:          cc.TransitEncryptionEnabled,
					keyAutoMinorVersionUpgrade:   cc.AutoMinorVersionUpgrade,
					"PreferredMaintenanceWindow": cc.PreferredMaintenanceWindow,
					keySnapshotRetentionLimit:    cc.SnapshotRetentionLimit,
					keySnapshotWindow:            cc.SnapshotWindow,
					keyCreateTime:                cc.CacheClusterCreateTime,
					keyStatus:                    cc.CacheClusterStatus,
				},
			}))
		}
	}

	return resources, nil
}

// GetColumns returns the CSV columns for the collector.
func (*ElastiCacheCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "SubCategory2", Value: func(r Resource) string { return r.SubCategory2 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		rawDataColumn("Description"),
		rawDataColumn(keyReplicationGroupID),
		rawDataColumn("ClusterID"),
		rawDataColumn("Engine"),
		rawDataColumn("Version"),
		rawDataColumn(keyNodeType),
		rawDataColumn("NodeGroups"),
		rawDataColumn("NumNodes"),
		rawDataColumn("CacheParameterGroup"),
		rawDataColumn("SecurityGroup"),
		rawDataColumn("MultiAZ"),
		rawDataColumn("AutomaticFailover"),
		rawDataColumn(keyEncryptedAtRest),
		rawDataColumn(keyEncryptedTransit),
		rawDataColumn(keyAuthTokenEnabled),
		rawDataColumn(keyAutoMinorVersionUpgrade),
		rawDataColumn("PreferredMaintenanceWindow"),
		rawDataColumn(keySnapshotRetentionLimit),
		rawDataColumn(keySnapshotWindow),
		rawDataColumn(keyStatus),
		rawDataColumn(keyCreateTime),
	}
}

// Name returns the resource name of the collector.
func (*ElastiCacheCollector) Name() string {
	return categoryElastiCache
}

// ShouldSort returns whether the collected resources should be sorted.
func (*ElastiCacheCollector) ShouldSort() bool {
	return false
}
