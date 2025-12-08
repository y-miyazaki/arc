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

// EC2Collector collects EC2 resources including instances, VPCs, and subnets
type EC2Collector struct{}

// Name returns the resource name of the collector.
func (*EC2Collector) Name() string {
	return "ec2"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*EC2Collector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
// This defines the output format for EC2 instance data
func (*EC2Collector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string {
			if r.Name == "" {
				return "N/A"
			}
			return r.Name
		}},
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

// Collect collects EC2 resources from AWS
// This includes EC2 instances with their associated VPC, subnet, and security group information
// The method performs the following steps:
// 1. Describe all EC2 instances in the region
// 2. Collect VPC and subnet IDs for name resolution
// 3. Resolve VPC and subnet names from tags
// 4. Process each instance and create resource entries
func (*EC2Collector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Create EC2 service client for the specified region
	svc := ec2.NewFromConfig(*cfg, func(o *ec2.Options) {
		o.Region = region
	})

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

		// Extract instance information
		name := helpers.GetTagValue(instance.Tags, tagNameKey)

		// Resolve VPC and Subnet names
		vpcName := helpers.ResolveNameFromMap(instance.VpcId, vpcNames)
		subnetName := helpers.ResolveNameFromMap(instance.SubnetId, subnetNames)

		// Security Groups - collect security group names
		var sgNames []string
		for _, sg := range instance.SecurityGroups {
			sgNames = append(sgNames, helpers.StringValue(sg.GroupName))
		}
		// Create resource entry for this instance
		resources = append(resources, NewResource(&ResourceInput{
			Category:    "ec2",
			SubCategory: "Instance",
			Name:        name,
			Region:      region,
			RawData: map[string]any{
				"InstanceID":    instance.InstanceId,
				"InstanceType":  instance.InstanceType,
				"ImageID":       instance.ImageId,
				"VPC":           vpcName,
				"Subnet":        subnetName,
				"SecurityGroup": sgNames,
				"State":         instance.State.Name,
			},
		}))
	}

	return resources, nil
}
