package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SQSCollector collects SQS queues.
type SQSCollector struct{}

// Name returns the collector name.
func (*SQSCollector) Name() string {
	return "sqs"
}

// ShouldSort returns true.
func (*SQSCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for SQS queues.
func (*SQSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "DelaySeconds", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DelaySeconds") }},
		{Header: "MaximumMessageSize", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MaximumMessageSize") }},
		{Header: "MessageRetentionPeriod", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MessageRetentionPeriod") }},
		{Header: "ReceiveMessageWaitTimeSeconds", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ReceiveMessageWaitTimeSeconds") }},
		{Header: "VisibilityTimeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "VisibilityTimeout") }},
		{Header: "RedrivePolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RedrivePolicy") }},
		{Header: "CreatedTimestamp", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedTimestamp") }},
		{Header: "LastModifiedTimestamp", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastModifiedTimestamp") }},
	}
}

// Collect collects SQS queues from the specified region.
func (*SQSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := sqs.NewFromConfig(*cfg, func(o *sqs.Options) {
		o.Region = region
	})

	var resources []Resource

	// List queues
	paginator := sqs.NewListQueuesPaginator(svc, &sqs.ListQueuesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list SQS queues: %w", err)
		}

		for _, qURL := range page.QueueUrls {
			// Extract name from URL (https://sqs.region.amazonaws.com/account/name)
			name := qURL
			if idx := strings.LastIndex(qURL, "/"); idx != -1 {
				name = qURL[idx+1:]
			}

			// Get queue attributes
			var attrOut *sqs.GetQueueAttributesOutput
			attrOut, err = svc.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
				QueueUrl: aws.String(qURL),
				AttributeNames: []types.QueueAttributeName{
					types.QueueAttributeNameAll,
				},
			})
			if err != nil {
				continue
			}
			attrs := attrOut.Attributes

			// Format RedrivePolicy JSON if present
			redrivePolicy := attrs["RedrivePolicy"]
			if formatted, formatErr := helpers.FormatJSONIndent(redrivePolicy); formatErr == nil {
				redrivePolicy = formatted
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "sqs",
				SubCategory: "Queue",
				Name:        name,
				Region:      region,
				ARN:         attrs["QueueArn"], // ARN is in attributes
				RawData: map[string]any{
					"DelaySeconds":                  attrs["DelaySeconds"],
					"MaximumMessageSize":            attrs["MaximumMessageSize"],
					"MessageRetentionPeriod":        attrs["MessageRetentionPeriod"],
					"ReceiveMessageWaitTimeSeconds": attrs["ReceiveMessageWaitTimeSeconds"],
					"VisibilityTimeout":             attrs["VisibilityTimeout"],
					"RedrivePolicy":                 redrivePolicy,
					"CreatedTimestamp":              attrs["CreatedTimestamp"],
					"LastModifiedTimestamp":         attrs["LastModifiedTimestamp"],
				},
			}))
		}
	}

	return resources, nil
}
