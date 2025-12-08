// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// EFSCollector collects EFS resources.
// It uses dependency injection to manage EFS clients for multiple regions.
// It collects File Systems, Mount Targets, and Access Points.
type EFSCollector struct {
	clients      map[string]*efs.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewEFSCollector creates a new EFS collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create EFS clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *EFSCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewEFSCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*EFSCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *efs.Client {
		return efs.NewFromConfig(*c, func(o *efs.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create EFS clients: %w", err)
	}

	return &EFSCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the collector name.
func (*EFSCollector) Name() string {
	return "efs"
}

// ShouldSort returns true.
func (*EFSCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for EFS resources.
func (*EFSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Performance", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Performance") }},
		{Header: "Throughput", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Throughput") }},
		{Header: "Encrypted", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Encrypted") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "Size", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Size") }},
		{Header: "Subnet", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Subnet") }},
		{Header: "IPAddress", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "IPAddress") }},
		{Header: "SecurityGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityGroup") }},
		{Header: "Path", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Path") }},
		{Header: "UID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "UID") }},
		{Header: "GID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "GID") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
		{Header: "CreationTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationTime") }},
	}
}

// Collect collects EFS resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *EFSCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
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
	// Get all KMS keys to resolve names efficiently
	kmsKeyMap, err := c.nameResolver.GetAllKMSKeys(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}

	// Get all subnets to resolve names efficiently
	subnetMap, err := c.nameResolver.GetAllSubnets(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnets: %w", err)
	}

	// Describe File Systems
	paginator := efs.NewDescribeFileSystemsPaginator(svc, &efs.DescribeFileSystemsInput{})
	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to describe file systems: %w", pageErr)
		}

		for i := range page.FileSystems {
			fs := &page.FileSystems[i]
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "efs",
				SubCategory: "FileSystem",
				Name:        fs.Name,
				Region:      region,
				ARN:         fs.FileSystemId,
				RawData: map[string]any{
					"Type":         "FileSystem",
					"Performance":  fs.PerformanceMode,
					"Throughput":   fs.ThroughputMode,
					"Encrypted":    fs.Encrypted,
					"KmsKey":       helpers.ResolveNameFromMap(fs.KmsKeyId, kmsKeyMap),
					"Size":         fs.SizeInBytes.Value,
					"State":        fs.LifeCycleState,
					"CreationTime": fs.CreationTime,
				},
			}))

			// Describe Mount Targets
			mtPaginator := efs.NewDescribeMountTargetsPaginator(svc, &efs.DescribeMountTargetsInput{
				FileSystemId: fs.FileSystemId,
			})
			for mtPaginator.HasMorePages() {
				mtPage, mtErr := mtPaginator.NextPage(ctx)
				if mtErr != nil {
					continue
				}
				for j := range mtPage.MountTargets {
					mt := &mtPage.MountTargets[j]
					// Get Security Groups
					sgOut, sgErr := svc.DescribeMountTargetSecurityGroups(ctx, &efs.DescribeMountTargetSecurityGroupsInput{
						MountTargetId: mt.MountTargetId,
					})
					var sgIDs []*string
					if sgErr == nil {
						for k := range sgOut.SecurityGroups {
							sgIDs = append(sgIDs, &sgOut.SecurityGroups[k])
						}
					}
					resources = append(resources, NewResource(&ResourceInput{
						Category:    "efs",
						SubCategory: "MountTarget",
						Name:        mt.MountTargetId,
						Region:      region,
						ARN:         mt.MountTargetId,
						RawData: map[string]any{
							"Type":          "MountTarget",
							"Subnet":        helpers.ResolveNameFromMap(mt.SubnetId, subnetMap),
							"IPAddress":     mt.IpAddress,
							"SecurityGroup": helpers.ResolveNamesFromMap(sgIDs, sgMap),
							"State":         mt.LifeCycleState,
						},
					}))
				}
			}

			// Describe Access Points
			apPaginator := efs.NewDescribeAccessPointsPaginator(svc, &efs.DescribeAccessPointsInput{
				FileSystemId: fs.FileSystemId,
			})
			for apPaginator.HasMorePages() {
				apPage, apErr := apPaginator.NextPage(ctx)
				if apErr != nil {
					continue
				}
				for j := range apPage.AccessPoints {
					ap := &apPage.AccessPoints[j]
					var uid, gid *int64
					if ap.PosixUser != nil {
						uid = ap.PosixUser.Uid
						gid = ap.PosixUser.Gid
					}
					path := "/"
					if ap.RootDirectory != nil && ap.RootDirectory.Path != nil {
						path = *ap.RootDirectory.Path
					}

					resources = append(resources, NewResource(&ResourceInput{
						Category:    "efs",
						SubCategory: "AccessPoint",
						Name:        ap.Name,
						Region:      region,
						ARN:         ap.AccessPointId,
						RawData: map[string]any{
							"Type":  "AccessPoint",
							"Path":  path,
							"UID":   uid,
							"GID":   gid,
							"State": ap.LifeCycleState,
						},
					}))
				}
			}
		}
	}

	return resources, nil
}
