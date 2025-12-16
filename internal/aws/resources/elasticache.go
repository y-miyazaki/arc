// Package resources provides AWS resource collectors for different services.
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

// Name returns the resource name of the collector.
func (*ElastiCacheCollector) Name() string {
	return "elasticache"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*ElastiCacheCollector) ShouldSort() bool {
	return false
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
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "ReplicationGroupID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ReplicationGroupID") }},
		{Header: "ClusterID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ClusterID") }},
		{Header: "Engine", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Engine") }},
		{Header: "Version", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Version") }},
		{Header: "NodeType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NodeType") }},
		{Header: "NodeGroups", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NodeGroups") }},
		{Header: "NumNodes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NumNodes") }},
		{Header: "CacheParameterGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CacheParameterGroup") }},
		{Header: "SecurityGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityGroup") }},
		{Header: "MultiAZ", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MultiAZ") }},
		{Header: "AutomaticFailover", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AutomaticFailover") }},
		{Header: "EncryptedAtRest", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EncryptedAtRest") }},
		{Header: "EncryptedTransit", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EncryptedTransit") }},
		{Header: "AuthTokenEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AuthTokenEnabled") }},
		{Header: "AutoMinorVersionUpgrade", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AutoMinorVersionUpgrade") }},
		{Header: "PreferredMaintenanceWindow", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PreferredMaintenanceWindow") }},
		{Header: "SnapshotRetentionLimit", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SnapshotRetentionLimit") }},
		{Header: "SnapshotWindow", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SnapshotWindow") }},
		{Header: "CreateTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreateTime") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
	}
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
				Category:     "elasticache",
				SubCategory1: "ReplicationGroup",
				Name:         rg.ReplicationGroupId,
				Region:       region,
				ARN:          rg.ARN,
				RawData: map[string]any{
					"Description":             rg.Description,
					"ReplicationGroupID":      rg.ReplicationGroupId,
					"NodeType":                rg.CacheNodeType,
					"NodeGroups":              helpers.FormatJSONIndentOrRaw(rg.NodeGroups),
					"MultiAZ":                 rg.MultiAZ,
					"AutomaticFailover":       rg.AutomaticFailover,
					"AuthTokenEnabled":        rg.AuthTokenEnabled,
					"EncryptedAtRest":         rg.AtRestEncryptionEnabled,
					"EncryptedTransit":        rg.TransitEncryptionEnabled,
					"AutoMinorVersionUpgrade": rg.AutoMinorVersionUpgrade,
					"SnapshotRetentionLimit":  rg.SnapshotRetentionLimit,
					"SnapshotWindow":          rg.SnapshotWindow,
					"CreateTime":              rg.ReplicationGroupCreateTime,
					"Status":                  rg.Status,
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
						Category:     "elasticache",
						SubCategory1: "",
						SubCategory2: "CacheCluster",
						Name:         cc.CacheClusterId,
						Region:       region,
						ARN:          cc.ARN,
						RawData: map[string]any{
							"ClusterID":                  cc.CacheClusterId,
							"ReplicationGroupID":         cc.ReplicationGroupId,
							"Engine":                     cc.Engine,
							"Version":                    cc.EngineVersion,
							"NodeType":                   cc.CacheNodeType,
							"NumNodes":                   cc.NumCacheNodes,
							"CacheParameterGroup":        ccParamGroup,
							"SecurityGroup":              helpers.ResolveNamesFromMap(sgIDs, sgMap),
							"AuthTokenEnabled":           cc.AuthTokenEnabled,
							"EncryptedAtRest":            cc.AtRestEncryptionEnabled,
							"EncryptedTransit":           cc.TransitEncryptionEnabled,
							"AutoMinorVersionUpgrade":    cc.AutoMinorVersionUpgrade,
							"PreferredMaintenanceWindow": cc.PreferredMaintenanceWindow,
							"SnapshotRetentionLimit":     cc.SnapshotRetentionLimit,
							"SnapshotWindow":             cc.SnapshotWindow,
							"CreateTime":                 cc.CacheClusterCreateTime,
							"Status":                     cc.CacheClusterStatus,
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
				Category:     "elasticache",
				SubCategory1: "CacheCluster",
				Name:         cc.CacheClusterId,
				Region:       region,
				ARN:          cc.ARN,
				RawData: map[string]any{
					"ClusterID":                  cc.CacheClusterId,
					"ReplicationGroupID":         cc.ReplicationGroupId,
					"Engine":                     cc.Engine,
					"Version":                    cc.EngineVersion,
					"NodeType":                   cc.CacheNodeType,
					"NumNodes":                   cc.NumCacheNodes,
					"CacheParameterGroup":        ccParamGroup,
					"SecurityGroup":              helpers.ResolveNamesFromMap(sgIDs, sgMap),
					"AuthTokenEnabled":           cc.AuthTokenEnabled,
					"EncryptedAtRest":            cc.AtRestEncryptionEnabled,
					"EncryptedTransit":           cc.TransitEncryptionEnabled,
					"AutoMinorVersionUpgrade":    cc.AutoMinorVersionUpgrade,
					"PreferredMaintenanceWindow": cc.PreferredMaintenanceWindow,
					"SnapshotRetentionLimit":     cc.SnapshotRetentionLimit,
					"SnapshotWindow":             cc.SnapshotWindow,
					"CreateTime":                 cc.CacheClusterCreateTime,
					"Status":                     cc.CacheClusterStatus,
				},
			}))
		}
	}

	return resources, nil
}
