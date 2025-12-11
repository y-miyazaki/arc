// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// tagNameKey is the standard AWS tag key for resource names
const tagNameKey = "Name"

// EC2Collector collects EC2 resources including instances, VPCs, and subnets.
// It uses dependency injection to manage EC2 clients for multiple regions.
type EC2Collector struct {
	clients      map[string]*ec2.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewEC2Collector creates a new EC2 collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create EC2 clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *EC2Collector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewEC2Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*EC2Collector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *ec2.Client {
		return ec2.NewFromConfig(*c, func(o *ec2.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 clients: %w", err)
	}

	return &EC2Collector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*EC2Collector) Name() string {
	return "ec2"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*EC2Collector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*EC2Collector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "InstanceID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "InstanceID") }},
		{Header: "InstanceType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "InstanceType") }},
		{Header: "ImageID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ImageID") }},
		{Header: "VPC", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "VPC") }},
		{Header: "Subnet", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Subnet") }},
		{Header: "SecurityGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityGroup") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

// Collect collects EC2 resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *EC2Collector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	// Initialize data structures for collecting resources
	var resources []Resource
	var instances []types.Instance
	vpcIDs := make(map[string]bool)
	subnetIDs := make(map[string]bool)

	// Describe Instances - collect all EC2 instances in the region
	paginator := ec2.NewDescribeInstancesPaginator(svc, &ec2.DescribeInstancesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}

		// Collect VPC and subnet IDs for name resolution
		for i := range page.Reservations {
			reservation := &page.Reservations[i]
			instances = append(instances, reservation.Instances...)
			for j := range reservation.Instances {
				instance := &reservation.Instances[j]
				if instance.VpcId != nil {
					vpcIDs[*instance.VpcId] = true
				}
				if instance.SubnetId != nil {
					subnetIDs[*instance.SubnetId] = true
				}
			}
		}
	}

	// Resolve VPC Names - get human-readable names for VPCs
	vpcNames := make(map[string]string)
	if len(vpcIDs) > 0 {
		var ids []string
		for id := range vpcIDs {
			ids = append(ids, id)
		}
		// Describe VPCs in batches if needed, but for now assume small number
		vpcOut, err := svc.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			VpcIds: ids,
		})
		if err == nil {
			for i := range vpcOut.Vpcs {
				vpc := &vpcOut.Vpcs[i]
				name := helpers.GetTagValue(vpc.Tags, tagNameKey)
				if name != "" {
					vpcNames[*vpc.VpcId] = name
				} else {
					vpcNames[*vpc.VpcId] = *vpc.VpcId
				}
			}
		}
	}

	// Resolve Subnet Names - get human-readable names for subnets
	subnetNames := make(map[string]string)
	if len(subnetIDs) > 0 {
		var ids []string
		for id := range subnetIDs {
			ids = append(ids, id)
		}
		subnetOut, err := svc.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: ids,
		})
		if err == nil {
			for i := range subnetOut.Subnets {
				subnet := &subnetOut.Subnets[i]
				name := helpers.GetTagValue(subnet.Tags, tagNameKey)
				if name != "" {
					subnetNames[*subnet.SubnetId] = name
				} else {
					subnetNames[*subnet.SubnetId] = *subnet.SubnetId
				}
			}
		}
	}

	// Process Instances - create resource entries for each instance
	for i := range instances {
		instance := &instances[i]

		// Security Groups - collect security group names
		var sgNames []string
		for _, sg := range instance.SecurityGroups {
			sgNames = append(sgNames, helpers.StringValue(sg.GroupName))
		}
		// Create resource entry for this instance
		resources = append(resources, NewResource(&ResourceInput{
			Category:     "ec2",
			SubCategory1: "Instance",
			Name:         helpers.GetTagValue(instance.Tags, tagNameKey),
			Region:       region,
			RawData: map[string]any{
				"InstanceID":    instance.InstanceId,
				"InstanceType":  instance.InstanceType,
				"ImageID":       instance.ImageId,
				"VPC":           helpers.ResolveNameFromMap(instance.VpcId, vpcNames),
				"Subnet":        helpers.ResolveNameFromMap(instance.SubnetId, subnetNames),
				"SecurityGroup": sgNames,
				"State":         instance.State.Name,
			},
		}))
	}

	return resources, nil
}
