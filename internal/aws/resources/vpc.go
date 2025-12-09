// Package resources provides AWS resource collectors.
//
//nolint:revive // comments-density: VPC collector has many API calls, additional comments would be redundant
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// VPCCollector collects VPC resources.
// It uses dependency injection to manage EC2 clients for multiple regions.
type VPCCollector struct {
	clients      map[string]*ec2.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewVPCCollector creates a new VPC collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create EC2 clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *VPCCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewVPCCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*VPCCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *ec2.Client {
		return ec2.NewFromConfig(*c, func(o *ec2.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 clients: %w", err)
	}

	return &VPCCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*VPCCollector) Name() string {
	return "vpc"
}

// ShouldSort returns whether the collected resources should be sorted.
// VPC should not be sorted to maintain parent-child order
func (*VPCCollector) ShouldSort() bool {
	return false
}

// GetColumns returns the CSV columns for the collector.
func (*VPCCollector) GetColumns() []Column {
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
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "CIDR", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CIDR") }},
		{Header: "PublicIP", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PublicIP") }},
		{Header: "Inbound", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Inbound") }},
		{Header: "Outbound", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Outbound") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Service", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Service") }},
		{Header: "Subnets", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Subnets") }},
		{Header: "RouteTables", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RouteTables") }},
		{Header: "SecurityGroups", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityGroups") }},
		{Header: "Settings", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Settings") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

// Collect collects VPC resources for the specified region.
// The collector must have been initialized with a client for this region.
//
//nolint:revive // cognitive-complexity: VPC collector inherently complex due to many subresources
func (c *VPCCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Get all VPCs
	vpcsOut, err := svc.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe vpcs: %w", err)
	}

	for i := range vpcsOut.Vpcs {
		vpc := &vpcsOut.Vpcs[i]

		// Add VPC resource
		resources = append(resources, NewResource(&ResourceInput{
			Category:    "vpc",
			SubCategory: "VPC",
			Name:        helpers.GetTagValue(vpc.Tags, "Name"),
			Region:      region,
			RawData: map[string]any{
				"ID":    vpc.VpcId,
				"CIDR":  vpc.CidrBlock,
				"State": vpc.State,
			},
		}))

		// Get route tables for this VPC
		rtOut, rtErr := svc.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			Filters: []types.Filter{{Name: aws.String("vpc-id"), Values: []string{aws.ToString(vpc.VpcId)}}},
		})
		if rtErr != nil {
			return nil, fmt.Errorf("failed to describe route tables: %w", rtErr)
		}

		// Build route table to IGW mapping for subnet classification
		rtHasIGW := make(map[string]bool)
		var mainRTID string
		for j := range rtOut.RouteTables {
			rt := &rtOut.RouteTables[j]
			rtID := helpers.StringValue(rt.RouteTableId)

			// Check if main route table
			for k := range rt.Associations {
				if aws.ToBool(rt.Associations[k].Main) {
					mainRTID = rtID
					break
				}
			}

			// Check if has IGW route
			for k := range rt.Routes {
				gwID := helpers.StringValue(rt.Routes[k].GatewayId)
				if strings.HasPrefix(gwID, "igw-") {
					rtHasIGW[rtID] = true
					break
				}
			}
		}

		// Build subnet to route table mapping
		subnetToRT := make(map[string]string)
		for j := range rtOut.RouteTables {
			rt := &rtOut.RouteTables[j]
			rtID := helpers.StringValue(rt.RouteTableId)
			for k := range rt.Associations {
				if rt.Associations[k].SubnetId != nil {
					subnetToRT[helpers.StringValue(rt.Associations[k].SubnetId)] = rtID
				}
			}
		}

		// Get subnets for this VPC
		subnetsOut, subErr := svc.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			Filters: []types.Filter{{Name: aws.String("vpc-id"), Values: []string{aws.ToString(vpc.VpcId)}}},
		})
		if subErr != nil {
			return nil, fmt.Errorf("failed to describe subnets: %w", subErr)
		}

		var publicSubnets, privateSubnets []Resource
		for j := range subnetsOut.Subnets {
			subnet := &subnetsOut.Subnets[j]

			// Determine if public or private
			rtID := subnetToRT[helpers.StringValue(subnet.SubnetId)]
			if rtID == "" {
				rtID = mainRTID
			}
			isPublic := rtHasIGW[rtID]

			subCategory := "PrivateSubnet"
			if isPublic {
				subCategory = "PublicSubnet"
			}

			subnetResource := NewResource(&ResourceInput{
				Category:       "vpc",
				SubSubCategory: subCategory,
				Name:           helpers.GetTagValue(subnet.Tags, "Name"),
				Region:         region,
				RawData: map[string]any{
					"ID":   subnet.SubnetId,
					"CIDR": subnet.CidrBlock,
				},
			})

			if isPublic {
				publicSubnets = append(publicSubnets, subnetResource)
			} else {
				privateSubnets = append(privateSubnets, subnetResource)
			}
		}

		// Add public subnets first, then private
		resources = append(resources, publicSubnets...)
		resources = append(resources, privateSubnets...)

		// Add route tables
		for j := range rtOut.RouteTables {
			rt := &rtOut.RouteTables[j]

			resources = append(resources, NewResource(&ResourceInput{
				Category:       "vpc",
				SubSubCategory: "RouteTable",
				Name:           helpers.GetTagValue(rt.Tags, "Name"),
				Region:         region,
				RawData: map[string]any{
					"ID": rt.RouteTableId,
				},
			}))
		}

		// Get Internet Gateways
		igwOut, igwErr := svc.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
			Filters: []types.Filter{{Name: aws.String("attachment.vpc-id"), Values: []string{aws.ToString(vpc.VpcId)}}},
		})
		if igwErr != nil {
			return nil, fmt.Errorf("failed to describe internet gateways: %w", igwErr)
		}

		for j := range igwOut.InternetGateways {
			igw := &igwOut.InternetGateways[j]

			resources = append(resources, NewResource(&ResourceInput{
				Category:       "vpc",
				SubSubCategory: "InternetGateway",
				Name:           helpers.GetTagValue(igw.Tags, "Name"),
				Region:         region,
				RawData: map[string]any{
					"ID":    igw.InternetGatewayId,
					"State": "attached",
				},
			}))
		}

		// Get NAT Gateways
		natOut, natErr := svc.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
			Filter: []types.Filter{{Name: aws.String("vpc-id"), Values: []string{aws.ToString(vpc.VpcId)}}},
		})
		if natErr != nil {
			return nil, fmt.Errorf("failed to describe nat gateways: %w", natErr)
		}

		for j := range natOut.NatGateways {
			nat := &natOut.NatGateways[j]

			// Collect all public IPs (primary and secondary)
			var publicIPs []string
			for k := range nat.NatGatewayAddresses {
				addr := &nat.NatGatewayAddresses[k]
				ip := helpers.StringValue(addr.PublicIp)
				if ip != "" && ip != "N/A" {
					isPrimary := ""
					if addr.IsPrimary != nil && *addr.IsPrimary {
						isPrimary = " (Primary)"
					}
					publicIPs = append(publicIPs, ip+isPrimary)
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:       "vpc",
				SubSubCategory: "NATGateway",
				Name:           helpers.GetTagValue(nat.Tags, "Name"),
				Region:         region,
				RawData: map[string]any{
					"ID":       nat.NatGatewayId,
					"PublicIP": publicIPs,
					"State":    nat.State,
				},
			}))
		}

		// Get Network ACLs
		naclOut, naclErr := svc.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{
			Filters: []types.Filter{{Name: aws.String("vpc-id"), Values: []string{aws.ToString(vpc.VpcId)}}},
		})
		if naclErr != nil {
			return nil, fmt.Errorf("failed to describe network acls: %w", naclErr)
		}

		for j := range naclOut.NetworkAcls {
			nacl := &naclOut.NetworkAcls[j]

			// Format entries
			var entries []string
			for k := range nacl.Entries {
				entry := &nacl.Entries[k]
				portRange := "-"
				if entry.PortRange != nil {
					portRange = fmt.Sprintf("%d-%d", aws.ToInt32(entry.PortRange.From), aws.ToInt32(entry.PortRange.To))
				}
				entryStr := fmt.Sprintf("Rule#: %d | Protocol: %s | RuleAction: %s | Egress: %t | CIDR: %s | PortRange: %s",
					aws.ToInt32(entry.RuleNumber),
					helpers.StringValue(entry.Protocol),
					string(entry.RuleAction),
					aws.ToBool(entry.Egress),
					helpers.StringValue(entry.CidrBlock),
					portRange)
				entries = append(entries, entryStr)
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:       "vpc",
				SubSubCategory: "NetworkACL",
				Name:           helpers.GetTagValue(nacl.Tags, "Name"),
				Region:         region,
				RawData: map[string]any{
					"ID":       nacl.NetworkAclId,
					"Settings": entries,
				},
			}))
		}

		// Get Security Groups
		sgOut, sgErr := svc.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			Filters: []types.Filter{{Name: aws.String("vpc-id"), Values: []string{aws.ToString(vpc.VpcId)}}},
		})
		if sgErr != nil {
			return nil, fmt.Errorf("failed to describe security groups: %w", sgErr)
		}

		for j := range sgOut.SecurityGroups {
			sg := &sgOut.SecurityGroups[j]

			// Format inbound rules
			var inbound []string
			for k := range sg.IpPermissions {
				perm := &sg.IpPermissions[k]
				for l := range perm.IpRanges {
					inbound = append(inbound, fmt.Sprintf("Protocol: %s | FromPort: %d | ToPort: %d | CIDR: %s",
						helpers.StringValue(perm.IpProtocol),
						aws.ToInt32(perm.FromPort),
						aws.ToInt32(perm.ToPort),
						helpers.StringValue(perm.IpRanges[l].CidrIp)))
				}
			}

			// Format outbound rules
			var outbound []string
			for k := range sg.IpPermissionsEgress {
				perm := &sg.IpPermissionsEgress[k]
				for l := range perm.IpRanges {
					outbound = append(outbound, fmt.Sprintf("Protocol: %s | FromPort: %d | ToPort: %d | CIDR: %s",
						helpers.StringValue(perm.IpProtocol),
						aws.ToInt32(perm.FromPort),
						aws.ToInt32(perm.ToPort),
						helpers.StringValue(perm.IpRanges[l].CidrIp)))
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:       "vpc",
				SubSubCategory: "SecurityGroup",
				Name:           sg.GroupName,
				Region:         region,
				RawData: map[string]any{
					"ID":          sg.GroupId,
					"Description": sg.Description,
					"Inbound":     inbound,
					"Outbound":    outbound,
				},
			}))
		}

		// Get VPC Endpoints
		epOut, epErr := svc.DescribeVpcEndpoints(ctx, &ec2.DescribeVpcEndpointsInput{
			Filters: []types.Filter{{Name: aws.String("vpc-id"), Values: []string{aws.ToString(vpc.VpcId)}}},
		})
		if epErr != nil {
			return nil, fmt.Errorf("failed to describe vpc endpoints: %w", epErr)
		}

		for j := range epOut.VpcEndpoints {
			ep := &epOut.VpcEndpoints[j]

			var sgIDs []string
			for _, g := range ep.Groups {
				sgIDs = append(sgIDs, helpers.StringValue(g.GroupId))
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:       "vpc",
				SubSubCategory: "Endpoint",
				Name:           helpers.GetTagValue(ep.Tags, "Name"),
				Region:         region,
				RawData: map[string]any{
					"ID":             ep.VpcEndpointId,
					"Type":           ep.VpcEndpointType,
					"Service":        ep.ServiceName,
					"Subnets":        ep.SubnetIds,
					"RouteTables":    ep.RouteTableIds,
					"SecurityGroups": sgIDs,
					"State":          ep.State,
				},
			}))
		}
	}

	return resources, nil
}
