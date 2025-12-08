package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
)

// CloudFrontCollector collects CloudFront resources.
type CloudFrontCollector struct{}

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
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
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

// Collect collects CloudFront resources.
func (*CloudFrontCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// CloudFront is a global service, only process from us-east-1
	if region != "us-east-1" {
		return nil, nil
	}

	svc := cloudfront.NewFromConfig(*cfg)
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
					Category:    "cloudfront",
					SubCategory: "Distribution",
					Name:        dist.DomainName,
					Region:      "Global",
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
