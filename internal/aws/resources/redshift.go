package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
)

// RedshiftCollector collects Redshift clusters.
// It retrieves cluster details, node types, and security groups.
// It also collects information about database names and endpoints.
// The collector uses the Redshift DescribeClusters API to list all clusters
// in the specified region.
// It implements the Collector interface.
type RedshiftCollector struct{}

// Name returns the collector name.
func (*RedshiftCollector) Name() string {
	return "redshift"
}

// ShouldSort returns true to enable sorting of results.
func (*RedshiftCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for Redshift.
// It defines headers and value extractors for cluster properties.
func (*RedshiftCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
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

// Collect collects Redshift clusters from the specified region.
// It retrieves cluster details including node types, VPCs, security groups, and KMS keys.
// Uses batch API calls to resolve names efficiently.
func (*RedshiftCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := redshift.NewFromConfig(*cfg, func(o *redshift.Options) {
		o.Region = region
	})

	var resources []Resource

	// Get all VPCs and KMS keys to resolve names efficiently
	vpcMap, err := helpers.GetAllVPCs(ctx, cfg, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPCs: %w", err)
	}
	kmsMap, err := helpers.GetAllKMSKeys(ctx, cfg, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}
	sgMap, err := helpers.GetAllSecurityGroups(ctx, cfg, region)
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

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "redshift",
				SubCategory: "Cluster",
				Name:        cluster.ClusterIdentifier,
				Region:      region,
				ARN:         roleARN,
				RawData: map[string]any{
					"NodeType":               cluster.NodeType,
					"NumberOfNodes":          cluster.NumberOfNodes,
					"DBName":                 cluster.DBName,
					"Endpoint":               endpoint,
					"Port":                   port,
					"MasterUsername":         cluster.MasterUsername,
					"VPCName":                helpers.ResolveNameFromMap(cluster.VpcId, vpcMap),
					"ClusterSubnetGroupName": cluster.ClusterSubnetGroupName,
					"SecurityGroup":          helpers.ResolveNamesFromMap(sgIDs, sgMap),
					"Encrypted":              cluster.Encrypted,
					"KmsKey":                 helpers.ResolveNameFromMap(cluster.KmsKeyId, kmsMap),
					"PubliclyAccessible":     cluster.PubliclyAccessible,
					"ClusterStatus":          cluster.ClusterStatus,
				},
			}))
		}
	}

	return resources, nil
}
