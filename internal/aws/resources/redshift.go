// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// RedshiftCollector collects Redshift clusters.
// It uses dependency injection to manage Redshift clients for multiple regions.
type RedshiftCollector struct {
	clients      map[string]*redshift.Client
	nameResolver *helpers.NameResolver
}

// NewRedshiftCollector creates a new Redshift collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Redshift clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *RedshiftCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewRedshiftCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*RedshiftCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *redshift.Client {
		return redshift.NewFromConfig(*c, func(o *redshift.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Redshift clients: %w", err)
	}

	return &RedshiftCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*RedshiftCollector) Name() string {
	return "redshift"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*RedshiftCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*RedshiftCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "RoleARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "NodeType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NodeType") }},
		{Header: "NumberOfNodes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NumberOfNodes") }},
		{Header: "DBName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DBName") }},
		{Header: "Endpoint", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Endpoint") }},
		{Header: "Port", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Port") }},
		{Header: "MasterUsername", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MasterUsername") }},
		{Header: "VPCName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "VPCName") }},
		{Header: "ClusterSubnetGroupName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ClusterSubnetGroupName") }},
		{Header: "SecurityGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityGroup") }},
		{Header: "Encrypted", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Encrypted") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "PubliclyAccessible", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PubliclyAccessible") }},
		{Header: "ClusterStatus", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ClusterStatus") }},
	}
}

// Collect collects Redshift resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *RedshiftCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Get resources for name resolution
	vpcs, err := c.nameResolver.GetAllVPCs(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPCs: %w", err)
	}
	kmsKeys, err := c.nameResolver.GetAllKMSKeys(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}
	securityGroups, err := c.nameResolver.GetAllSecurityGroups(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get security groups: %w", err)
	}

	paginator := redshift.NewDescribeClustersPaginator(svc, &redshift.DescribeClustersInput{})
	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to describe clusters: %w", pageErr)
		}

		for i := range page.Clusters {
			cluster := &page.Clusters[i]

			var roleARN string
			if len(cluster.IamRoles) > 0 {
				roleARN = aws.ToString(cluster.IamRoles[0].IamRoleArn)
			}

			var endpoint, port string
			if cluster.Endpoint != nil {
				endpoint = aws.ToString(cluster.Endpoint.Address)
				port = fmt.Sprintf("%d", *cluster.Endpoint.Port)
			}

			var sgIDs []*string
			for _, sg := range cluster.VpcSecurityGroups {
				sgIDs = append(sgIDs, sg.VpcSecurityGroupId)
			}

			r := NewResource(&ResourceInput{
				Category:     "redshift",
				SubCategory1: "Cluster",
				Name:         cluster.ClusterIdentifier,
				Region:       region,
				ARN:          roleARN,
				RawData: map[string]any{
					"NodeType":               cluster.NodeType,
					"NumberOfNodes":          cluster.NumberOfNodes,
					"DBName":                 cluster.DBName,
					"Endpoint":               endpoint,
					"Port":                   port,
					"MasterUsername":         cluster.MasterUsername,
					"VPCName":                helpers.ResolveNameFromMap(cluster.VpcId, vpcs),
					"ClusterSubnetGroupName": cluster.ClusterSubnetGroupName,
					"SecurityGroup":          helpers.ResolveNamesFromMap(sgIDs, securityGroups),
					"Encrypted":              cluster.Encrypted,
					"KmsKey":                 helpers.ResolveNameFromMap(cluster.KmsKeyId, kmsKeys),
					"PubliclyAccessible":     cluster.PubliclyAccessible,
					"ClusterStatus":          cluster.ClusterStatus,
				},
			})
			resources = append(resources, r)
		}
	}

	return resources, nil
}
