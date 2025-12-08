// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

const (
	// DefaultZeroString is the default string value for zero numeric values
	DefaultZeroString = "0"
)

// RDSCollector collects RDS resources.
// It uses dependency injection to manage RDS clients for multiple regions.
type RDSCollector struct {
	clients      map[string]*rds.Client
	nameResolver *helpers.NameResolver
}

// NewRDSCollector creates a new RDS collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create RDS clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *RDSCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewRDSCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*RDSCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *rds.Client {
		return rds.NewFromConfig(*c, func(o *rds.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS clients: %w", err)
	}

	return &RDSCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*RDSCollector) Name() string {
	return "rds"
}

// ShouldSort returns whether the collected resources should be sorted.
// RDS should not be sorted to maintain parent-child order (Cluster -> Instance)
func (*RDSCollector) ShouldSort() bool {
	return false
}

// GetColumns returns the CSV columns for the collector.
func (*RDSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Engine", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Engine") }},
		{Header: "Version", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Version") }},
		{Header: "InstanceClass", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "InstanceClass") }},
		{Header: "AllocatedStorage", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AllocatedStorage") }},
		{Header: "MultiAZ", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MultiAZ") }},
		{Header: "DBClusterMembers", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DBClusterMembers") }},
		{Header: "EngineLifecycleSupport", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EngineLifecycleSupport") }},
		{Header: "IAMDatabaseAuthenticationEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "IAMDatabaseAuthenticationEnabled") }},
		{Header: "KerberosAuth", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KerberosAuth") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "AvailabilityZone", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AvailabilityZone") }},
		{Header: "BackupRetentionPeriod", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "BackupRetentionPeriod") }},
	}
}

// Collect collects RDS resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *RDSCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Get KMS keys for name resolution
	kmsKeys, err := c.nameResolver.GetAllKMSKeys(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}

	// Track cluster member instances to avoid duplicates
	clusterMemberInstances := make(map[string]string) // instanceID -> clusterID|role

	// First, collect all clusters
	clustersOut, err := svc.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe db clusters: %w", err)
	}

	// Get all instances for later lookup
	instancesOut, instErr := svc.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if instErr != nil {
		return nil, fmt.Errorf("failed to describe db instances: %w", instErr)
	}

	// Build instance map for quick lookup
	instanceMap := make(map[string]*types.DBInstance)
	for i := range instancesOut.DBInstances {
		inst := &instancesOut.DBInstances[i]
		instanceMap[helpers.StringValue(inst.DBInstanceIdentifier)] = inst
	}

	// Process clusters
	for i := range clustersOut.DBClusters {
		cluster := &clustersOut.DBClusters[i]
		clusterID := helpers.StringValue(cluster.DBClusterIdentifier)

		// Kerberos Auth
		var kerberosAuth *bool
		for j := range cluster.DomainMemberships {
			if helpers.StringValue(cluster.DomainMemberships[j].Status) == "joined" {
				kerberosAuth = aws.Bool(true)
				break
			}
		}

		lifecycleSupport := helpers.StringValue(cluster.EngineLifecycleSupport)

		// Add cluster resource
		resources = append(resources, NewResource(&ResourceInput{
			Category:    "rds",
			SubCategory: "DBCluster",
			Name:        clusterID,
			Region:      region,
			RawData: map[string]any{
				"ID":                               clusterID,
				"Type":                             "DBCluster",
				"Engine":                           cluster.Engine,
				"Version":                          cluster.EngineVersion,
				"MultiAZ":                          cluster.MultiAZ,
				"DBClusterMembers":                 strconv.Itoa(len(cluster.DBClusterMembers)),
				"EngineLifecycleSupport":           cluster.EngineLifecycleSupport,
				"IAMDatabaseAuthenticationEnabled": cluster.IAMDatabaseAuthenticationEnabled,
				"KerberosAuth":                     kerberosAuth,
				"KmsKey":                           helpers.ResolveNameFromMap(cluster.KmsKeyId, kmsKeys),
				"AvailabilityZone":                 cluster.AvailabilityZones,
				"BackupRetentionPeriod":            cluster.BackupRetentionPeriod,
			},
		}))

		// Process cluster members
		for i := range cluster.DBClusterMembers {
			member := &cluster.DBClusterMembers[i]
			memberID := helpers.StringValue(member.DBInstanceIdentifier)
			role := "Reader"
			if aws.ToBool(member.IsClusterWriter) {
				role = "Writer"
			}
			clusterMemberInstances[memberID] = clusterID + "|" + role
		}

		// Get cluster member instances
		for j := range cluster.DBClusterMembers {
			member := &cluster.DBClusterMembers[j]
			memberID := helpers.StringValue(member.DBInstanceIdentifier)
			role := "Reader"
			if aws.ToBool(member.IsClusterWriter) {
				role = "Writer"
			}

			// Look up instance details
			if inst, ok2 := instanceMap[memberID]; ok2 {
				instLifecycleSupport := helpers.StringValue(inst.EngineLifecycleSupport)
				if instLifecycleSupport == "" {
					instLifecycleSupport = lifecycleSupport
				}

				resources = append(resources, NewResource(&ResourceInput{
					Category:       "rds",
					SubCategory:    "",
					SubSubCategory: "DBInstance",
					Name:           memberID,
					Region:         region,
					RawData: map[string]any{
						"ID":                               memberID,
						"Type":                             fmt.Sprintf("DBInstance (%s)", role),
						"Engine":                           inst.Engine,
						"Version":                          inst.EngineVersion,
						"InstanceClass":                    inst.DBInstanceClass,
						"AllocatedStorage":                 inst.AllocatedStorage,
						"MultiAZ":                          inst.MultiAZ,
						"EngineLifecycleSupport":           instLifecycleSupport,
						"IAMDatabaseAuthenticationEnabled": cluster.IAMDatabaseAuthenticationEnabled,
						"KerberosAuth":                     kerberosAuth,
						"KmsKey":                           helpers.ResolveNameFromMap(cluster.KmsKeyId, kmsKeys),
						"AvailabilityZone":                 inst.AvailabilityZone,
						"BackupRetentionPeriod":            cluster.BackupRetentionPeriod,
					},
				}))
			}
		}
	}

	// Process standalone instances (not part of any cluster)
	for i := range instancesOut.DBInstances {
		inst := &instancesOut.DBInstances[i]
		instID := helpers.StringValue(inst.DBInstanceIdentifier)
		clusterID := helpers.StringValue(inst.DBClusterIdentifier)

		// Skip if part of a cluster
		if clusterID != "" {
			continue
		}

		// Kerberos Auth
		var kerberosAuth *bool
		for j := range inst.DomainMemberships {
			if helpers.StringValue(inst.DomainMemberships[j].Status) == "joined" {
				kerberosAuth = aws.Bool(true)
				break
			}
		}

		resources = append(resources, NewResource(&ResourceInput{
			Category:    "rds",
			SubCategory: "DBInstance",
			Name:        instID,
			Region:      region,
			RawData: map[string]any{
				"ID":                               instID,
				"Type":                             "DBInstance",
				"Engine":                           inst.Engine,
				"Version":                          inst.EngineVersion,
				"InstanceClass":                    inst.DBInstanceClass,
				"AllocatedStorage":                 inst.AllocatedStorage,
				"MultiAZ":                          inst.MultiAZ,
				"EngineLifecycleSupport":           inst.EngineLifecycleSupport,
				"IAMDatabaseAuthenticationEnabled": inst.IAMDatabaseAuthenticationEnabled,
				"KerberosAuth":                     kerberosAuth,
				"KmsKey":                           helpers.ResolveNameFromMap(inst.KmsKeyId, kmsKeys),
				"AvailabilityZone":                 inst.AvailabilityZone,
				"BackupRetentionPeriod":            inst.BackupRetentionPeriod,
			},
		}))
	}

	return resources, nil
}
