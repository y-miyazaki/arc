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
type SNSCollector struct{}

// Name returns the collector name.
func (*SNSCollector) Name() string {
	return "sns"
}

// ShouldSort returns true.
func (*SNSCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for SNS topics.
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

// Collect collects SNS topics from the specified region.
func (*SNSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := sns.NewFromConfig(*cfg, func(o *sns.Options) {
		o.Region = region
	})

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

			resources = append(resources, NewResource(&ResourceInput{
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
			}))
		}
	}

	return resources, nil
}
