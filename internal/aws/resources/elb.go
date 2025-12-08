// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// ELBCollector collects ELB resources including load balancers, target groups, and listeners.
// It uses dependency injection to manage ELB, WAF, and EC2 clients for multiple regions.
type ELBCollector struct {
	elbClients   map[string]*elasticloadbalancingv2.Client
	wafClients   map[string]*wafv2.Client
	ec2Clients   map[string]*ec2.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewELBCollector creates a new ELB collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create ELB clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *ELBCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewELBCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*ELBCollector, error) {
	elbClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *elasticloadbalancingv2.Client {
		return elasticloadbalancingv2.NewFromConfig(*c, func(o *elasticloadbalancingv2.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ELB clients: %w", err)
	}

	wafClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *wafv2.Client {
		return wafv2.NewFromConfig(*c, func(o *wafv2.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create WAF clients: %w", err)
	}

	ec2Clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *ec2.Client {
		return ec2.NewFromConfig(*c, func(o *ec2.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create EC2 clients: %w", err)
	}

	return &ELBCollector{
		elbClients:   elbClients,
		wafClients:   wafClients,
		ec2Clients:   ec2Clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*ELBCollector) Name() string {
	return "elb"
}

// ShouldSort returns whether the collected resources should be sorted.
// ELB resources maintain grouping structure, so sorting is disabled
func (*ELBCollector) ShouldSort() bool {
	return false
}

// GetColumns returns the CSV columns for the collector.
// Defines the output format for ELB resource data including DNS, type, VPC, etc.
func (*ELBCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "DNSName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DNSName") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "VPC", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "VPC") }},
		{Header: "AvailabilityZone", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AvailabilityZone") }},
		{Header: "SecurityGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityGroup") }},
		{Header: "WAF", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "WAF") }},
		{Header: "Protocol", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Protocol") }},
		{Header: "Port", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Port") }},
		{Header: "HealthCheck", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "HealthCheck") }},
		{Header: "SSLPolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SSLPolicy") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
		{Header: "CreatedTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedTime") }},
	}
}

// Collect collects ELB resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *ELBCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.elbClients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	wafSvc, ok := c.wafClients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s (WAF)", ErrNoClientForRegion, region)
	}

	ec2Svc, ok := c.ec2Clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s (EC2)", ErrNoClientForRegion, region)
	}

	var resources []Resource
	var loadBalancers []elasticloadbalancingv2.DescribeLoadBalancersOutput

	// Get all security groups to resolve names efficiently
	sgNames, err := c.nameResolver.GetAllSecurityGroups(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get security groups: %w", err)
	}

	// Describe Load Balancers - fetch all load balancers in the region
	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(svc, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to describe load balancers: %w", pageErr)
		}
		loadBalancers = append(loadBalancers, *page)
	}

	// Collect VPC IDs for resolution - gather IDs to resolve names later
	vpcIDs := make(map[string]bool)
	for i := range loadBalancers {
		page := &loadBalancers[i]
		for j := range page.LoadBalancers {
			lb := &page.LoadBalancers[j]
			if lb.VpcId != nil {
				vpcIDs[*lb.VpcId] = true
			}
		}
	}

	// Resolve VPC Names - convert VPC IDs to human-readable names using tags
	vpcNames := make(map[string]string)
	if len(vpcIDs) > 0 {
		var ids []string
		for id := range vpcIDs {
			ids = append(ids, id)
		}
		vpcOut, vpcErr := ec2Svc.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{VpcIds: ids})
		if vpcErr == nil {
			for i := range vpcOut.Vpcs {
				vpc := &vpcOut.Vpcs[i]
				name := helpers.GetTagValue(vpc.Tags, "Name")
				if name != "" {
					vpcNames[*vpc.VpcId] = name
				} else {
					vpcNames[*vpc.VpcId] = *vpc.VpcId
				}
			}
		}
	}

	// Process Load Balancers - iterate through each load balancer and extract details
	for i := range loadBalancers {
		page := &loadBalancers[i]
		for j := range page.LoadBalancers {
			lb := &page.LoadBalancers[j]

			// Collect AZs and SGs
			var azs []string
			for k := range lb.AvailabilityZones {
				az := &lb.AvailabilityZones[k]
				azs = append(azs, helpers.StringValue(az.ZoneName))
			}

			var sgIDs []*string
			for _, sg := range lb.SecurityGroups {
				sgIDs = append(sgIDs, &sg)
			}

			// WAF Association - check if load balancer has WAF protection
			var lbWAF *string
			wafOut, wafErr := wafSvc.GetWebACLForResource(ctx, &wafv2.GetWebACLForResourceInput{
				ResourceArn: lb.LoadBalancerArn,
			})
			if wafErr == nil && wafOut.WebACL != nil {
				lbWAF = wafOut.WebACL.ARN
			}

			// Get load balancer state code with nil check
			var stateCode any
			if lb.State != nil {
				stateCode = lb.State.Code
			}

			// Add load balancer resource to results
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "elb",
				SubCategory: "LoadBalancer",
				Name:        lb.LoadBalancerName,
				Region:      region,
				ARN:         lb.LoadBalancerArn,
				RawData: map[string]any{
					"DNSName":          lb.DNSName,
					"Type":             lb.Type,
					"VPC":              helpers.ResolveNameFromMap(lb.VpcId, vpcNames),
					"AvailabilityZone": azs,
					"SecurityGroup":    helpers.ResolveNamesFromMap(sgIDs, sgNames),
					"WAF":              lbWAF,
					"State":            stateCode,
					"CreatedTime":      lb.CreatedTime,
				},
			}))

			// Target Groups - collect target groups associated with this load balancer
			tgPaginator := elasticloadbalancingv2.NewDescribeTargetGroupsPaginator(svc, &elasticloadbalancingv2.DescribeTargetGroupsInput{
				LoadBalancerArn: lb.LoadBalancerArn,
			})
			var tgPage *elasticloadbalancingv2.DescribeTargetGroupsOutput
			for tgPaginator.HasMorePages() {
				tgPage, err = tgPaginator.NextPage(ctx)
				if err != nil {
					continue
				}
				for k := range tgPage.TargetGroups {
					tg := &tgPage.TargetGroups[k]
					// Add target group resource to results
					resources = append(resources, NewResource(&ResourceInput{
						Category:       "elb",
						SubCategory:    "",
						SubSubCategory: "TargetGroup",
						Name:           tg.TargetGroupName,
						Region:         region,
						ARN:            tg.TargetGroupArn,
						RawData: map[string]any{
							"Type":        tg.TargetType,
							"Protocol":    tg.Protocol,
							"Port":        tg.Port,
							"HealthCheck": tg.HealthCheckPath,
						},
					}))
				}
			}

			// Listeners - collect listeners associated with this load balancer
			lsPaginator := elasticloadbalancingv2.NewDescribeListenersPaginator(svc, &elasticloadbalancingv2.DescribeListenersInput{
				LoadBalancerArn: lb.LoadBalancerArn,
			})
			var lsPage *elasticloadbalancingv2.DescribeListenersOutput
			for lsPaginator.HasMorePages() {
				lsPage, err = lsPaginator.NextPage(ctx)
				if err != nil {
					continue
				}
				for k := range lsPage.Listeners {
					ls := &lsPage.Listeners[k]
					name := fmt.Sprintf("%s:%d", ls.Protocol, aws.ToInt32(ls.Port))
					// Add listener resource to results
					resources = append(resources, NewResource(&ResourceInput{
						Category:       "elb",
						SubCategory:    "",
						SubSubCategory: "Listener",
						Name:           name,
						Region:         region,
						ARN:            ls.ListenerArn,
						RawData: map[string]any{
							"Protocol":  ls.Protocol,
							"Port":      ls.Port,
							"SSLPolicy": ls.SslPolicy,
						},
					}))
				}
			}
		}
	}

	return resources, nil
}
