// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// CloudFrontCollector collects CloudFront resources.
// It uses dependency injection to manage CloudFront clients for multiple regions.
type CloudFrontCollector struct {
	clients      map[string]*cloudfront.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewCloudFrontCollector creates a new CloudFront collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create CloudFront clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *CloudFrontCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewCloudFrontCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*CloudFrontCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *cloudfront.Client {
		return cloudfront.NewFromConfig(*c, func(o *cloudfront.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudFront clients: %w", err)
	}

	return &CloudFrontCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*CloudFrontCollector) Name() string {
	return "cloudfront"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*CloudFrontCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*CloudFrontCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "AlternateDomain", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AlternateDomain") }},
		{Header: "Origin", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Origin") }},
		{Header: "PriceClass", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PriceClass") }},
		{Header: "WAF", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "WAF") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
	}
}

// Collect collects CloudFront resources for the specified region.
// CloudFront is a global service, only process from us-east-1.
// The collector must have been initialized with a client for this region.
func (c *CloudFrontCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	// CloudFront is a global service, only process from us-east-1
	if region != "us-east-1" {
		return nil, nil
	}

	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	paginator := cloudfront.NewListDistributionsPaginator(svc, &cloudfront.ListDistributionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list distributions: %w", err)
		}

		if page.DistributionList != nil {
			for i := range page.DistributionList.Items {
				dist := &page.DistributionList.Items[i]

				// Aliases
				var aliases []string
				if dist.Aliases != nil {
					aliases = dist.Aliases.Items
				}

				// Origin
				var origin *string
				if dist.Origins != nil && len(dist.Origins.Items) > 0 {
					origin = dist.Origins.Items[0].DomainName
				}

				// WAF Name Resolution
				var waf *string
				if dist.WebACLId != nil && *dist.WebACLId != "" {
					wafARN := *dist.WebACLId
					// Try to extract the human-friendly name from WAFv2 ARN which contains "/webacl/<name>/"
					// Example: arn:aws:wafv2:us-east-1:123456789012:regional/webacl/MyWebACL/uuid
					if idx := strings.Index(wafARN, "/webacl/"); idx != -1 {
						// take the part after "/webacl/" and split by '/'
						sub := wafARN[idx+len("/webacl/"):]
						subParts := strings.SplitN(sub, "/", 2) //nolint:mnd
						if len(subParts) > 0 && subParts[0] != "" {
							wafName := subParts[0]
							waf = &wafName
						}
					}
					// Fallback: if we couldn't parse a name, try splitting on '/' and take the last segment
					if waf == nil {
						parts := strings.Split(wafARN, "/")
						if len(parts) > 0 {
							wafName := parts[len(parts)-1]
							waf = &wafName
						}
					}
				}

				resources = append(resources, NewResource(&ResourceInput{
					Category:     "cloudfront",
					SubCategory1: "Distribution",
					Name:         dist.DomainName,
					Region:       "Global",
					RawData: map[string]any{
						"ID":              dist.Id,
						"AlternateDomain": aliases,
						"Origin":          origin,
						"PriceClass":      dist.PriceClass,
						"WAF":             waf,
						"Status":          dist.Status,
					},
				}))
			}
		}
	}

	return resources, nil
}
