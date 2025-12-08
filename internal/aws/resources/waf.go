package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// WAFCollector collects WAFv2 WebACLs.
// It collects both Regional and CloudFront (Global) WebACLs.
// It retrieves details such as rules, associated resources, and logging configurations.
// The collector handles both regional and global scopes, using appropriate APIs for each.
// It also lists CloudFront distributions associated with global WebACLs.
type WAFCollector struct{}

// Name returns the collector name.
func (*WAFCollector) Name() string {
	return "waf"
}

// ShouldSort returns true.
func (*WAFCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for WAF.
func (*WAFCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
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

// Collect collects WAF resources from the specified region.
func (c *WAFCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	wafSvc := wafv2.NewFromConfig(*cfg, func(o *wafv2.Options) {
		o.Region = region
	})

	// CloudFront client for Global WAF associations (only needed if region is us-east-1)
	var cfSvc *cloudfront.Client
	if region == "us-east-1" {
		cfSvc = cloudfront.NewFromConfig(*cfg)
	}

	var resources []Resource

	// 1. Regional WAFs
	if err := c.collectScope(ctx, wafSvc, nil, region, types.ScopeRegional, &resources); err != nil {
		return nil, fmt.Errorf("failed to collect regional WAFs: %w", err)
	}

	// 2. CloudFront WAFs (Global) - only collect in us-east-1
	if region == "us-east-1" {
		if err := c.collectScope(ctx, wafSvc, cfSvc, "Global", types.ScopeCloudfront, &resources); err != nil {
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
				Category:    "waf",
				SubCategory: "WebACL",
				Name:        summary.Name,
				Region:      regionDesc,
				ARN:         summary.ARN,
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
