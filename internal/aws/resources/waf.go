// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// WAFCollector collects WAFv2 WebACLs.
// It uses dependency injection to manage WAFv2 and CloudFront clients.
// WAF is a global service for CloudFront - only processes from us-east-1 to avoid duplicates.
type WAFCollector struct {
	wafClient    map[string]*wafv2.Client
	cfClient     *cloudfront.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewWAFCollector creates a new WAF collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create WAF clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *WAFCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewWAFCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*WAFCollector, error) {
	wafClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *wafv2.Client {
		return wafv2.NewFromConfig(*c, func(o *wafv2.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create WAFv2 clients: %w", err)
	}

	// CloudFront client for Global WAF associations
	cfClient := cloudfront.NewFromConfig(*cfg)

	return &WAFCollector{
		wafClient:    wafClients,
		cfClient:     cfClient,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*WAFCollector) Name() string {
	return "waf"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*WAFCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*WAFCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "Scope", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Scope") }},
		{Header: "Rules", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Rules") }},
		{Header: "AssociatedResources", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AssociatedResources") }},
		{Header: "Logging", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Logging") }},
	}
}

// Collect collects WAF resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *WAFCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.wafClient[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// 1. Regional WAFs
	if err := c.collectScope(ctx, svc, nil, region, types.ScopeRegional, &resources); err != nil {
		return nil, fmt.Errorf("failed to collect regional WAFs: %w", err)
	}

	// 2. CloudFront WAFs (Global) - only collect in us-east-1
	if region == "us-east-1" {
		if err := c.collectScope(ctx, svc, c.cfClient, "Global", types.ScopeCloudfront, &resources); err != nil {
			return nil, fmt.Errorf("failed to collect global WAFs: %w", err)
		}
	}

	return resources, nil
}

func (*WAFCollector) collectScope(ctx context.Context, svc *wafv2.Client, cfSvc *cloudfront.Client, regionDesc string, scope types.Scope, resources *[]Resource) error {
	var nextMarker *string
	for {
		out, err := svc.ListWebACLs(ctx, &wafv2.ListWebACLsInput{
			Scope:      scope,
			NextMarker: nextMarker,
		})
		if err != nil {
			return fmt.Errorf("failed to list web ACLs for scope %s: %w", scope, err)
		}

		for i := range out.WebACLs {
			summary := &out.WebACLs[i]
			// Get WebACL details
			detail, detailErr := svc.GetWebACL(ctx, &wafv2.GetWebACLInput{
				Id:    summary.Id,
				Name:  summary.Name,
				Scope: scope,
			})
			if detailErr != nil {
				continue
			}

			var rules []string
			if detail.WebACL != nil {
				for j := range detail.WebACL.Rules {
					r := &detail.WebACL.Rules[j]
					rules = append(rules, aws.ToString(r.Name))
				}
			}

			// Associated Resources
			var associated []string
			if scope == types.ScopeRegional {
				rTypes := []types.ResourceType{
					types.ResourceTypeApplicationLoadBalancer,
					types.ResourceTypeApiGateway,
				}
				for _, rt := range rTypes {
					resOut, resErr := svc.ListResourcesForWebACL(ctx, &wafv2.ListResourcesForWebACLInput{
						WebACLArn:    summary.ARN,
						ResourceType: rt,
					})
					if resErr == nil {
						associated = append(associated, resOut.ResourceArns...)
					}
				}
			} else if scope == types.ScopeCloudfront && cfSvc != nil {
				cfOut, cfErr := cfSvc.ListDistributionsByWebACLId(ctx, &cloudfront.ListDistributionsByWebACLIdInput{
					WebACLId: summary.ARN,
				})
				if cfErr == nil && cfOut.DistributionList != nil {
					for k := range cfOut.DistributionList.Items {
						item := &cfOut.DistributionList.Items[k]
						associated = append(associated, aws.ToString(item.ARN))
					}
				}
			}

			// Logging Configuration
			var logging []string
			logOut, logErr := svc.GetLoggingConfiguration(ctx, &wafv2.GetLoggingConfigurationInput{
				ResourceArn: summary.ARN,
			})
			if logErr == nil && logOut.LoggingConfiguration != nil {
				logging = append(logging, logOut.LoggingConfiguration.LogDestinationConfigs...)
			}

			*resources = append(*resources, NewResource(&ResourceInput{
				Category:     "waf",
				SubCategory1: "WebACL",
				Name:         summary.Name,
				Region:       regionDesc,
				ARN:          summary.ARN,
				RawData: map[string]any{
					"Description":         summary.Description,
					"Scope":               string(scope),
					"Rules":               rules,
					"AssociatedResources": associated,
					"Logging":             logging,
				},
			}))
		}

		if out.NextMarker == nil {
			break
		}
		nextMarker = out.NextMarker
	}
	return nil
}
