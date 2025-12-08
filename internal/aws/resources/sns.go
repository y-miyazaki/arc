// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// SNSCollector collects SNS topics.
// It uses dependency injection to manage SNS clients for multiple regions.
type SNSCollector struct {
	clients      map[string]*sns.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewSNSCollector creates a new SNS collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create SNS clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *SNSCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewSNSCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*SNSCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *sns.Client {
		return sns.NewFromConfig(*c, func(o *sns.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create SNS clients: %w", err)
	}

	return &SNSCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*SNSCollector) Name() string {
	return "sns"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*SNSCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*SNSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "DisplayName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DisplayName") }},
		{Header: "Owner", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Owner") }},
		{Header: "Policy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Policy") }},
	}
}

// Collect collects SNS resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *SNSCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// List topics
	paginator := sns.NewListTopicsPaginator(svc, &sns.ListTopicsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list SNS topics: %w", err)
		}

		for _, topic := range page.Topics {
			if topic.TopicArn == nil {
				continue
			}
			arn := *topic.TopicArn
			// Extract name from ARN (arn:aws:sns:region:account:name)
			name := arn
			if idx := strings.LastIndex(arn, ":"); idx != -1 {
				name = arn[idx+1:]
			}

			// Get topic attributes
			var attrOut *sns.GetTopicAttributesOutput
			attrOut, err = svc.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
				TopicArn: topic.TopicArn,
			})
			if err != nil {
				// Skip if failed to get attributes (e.g. permission issue or deleted)
				continue
			}
			attrs := attrOut.Attributes

			// Format Policy as indented JSON
			policy := attrs["Policy"]
			if policyStr := attrs["Policy"]; policyStr != "" { // nolint: revive
				if formatted, errFormat := helpers.FormatJSONIndent(policyStr); errFormat == nil {
					policy = formatted
				}
			}

			r := NewResource(&ResourceInput{
				Category:    "sns",
				SubCategory: "Topic",
				Name:        name,
				Region:      region,
				ARN:         arn,
				RawData: map[string]any{
					"DisplayName": attrs["DisplayName"],
					"Owner":       attrs["Owner"],
					"Policy":      policy,
				},
			})
			resources = append(resources, r)
		}
	}

	return resources, nil
}
