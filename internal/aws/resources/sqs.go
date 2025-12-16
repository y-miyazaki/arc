package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Sentinel errors for SQS operations.
var (
	ErrNoSQSClient = errors.New("no SQS client found for region")
)

// SQSCollector collects SQS queues.
type SQSCollector struct {
	clients      map[string]*sqs.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewSQSCollector creates a new SQS collector with regional clients.
func NewSQSCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*SQSCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions,
		func(cfg *aws.Config, region string) *sqs.Client {
			return sqs.NewFromConfig(*cfg, func(o *sqs.Options) {
				o.Region = region
			})
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create SQS clients: %w", err)
	}

	return &SQSCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

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
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
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
func (c *SQSCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoSQSClient, region)
	}

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

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "sqs",
				SubCategory1: "Queue",
				Name:         name,
				Region:       region,
				ARN:          attrs["QueueArn"], // ARN is in attributes
				RawData: map[string]any{
					"DelaySeconds":                  attrs["DelaySeconds"],
					"MaximumMessageSize":            attrs["MaximumMessageSize"],
					"MessageRetentionPeriod":        attrs["MessageRetentionPeriod"],
					"ReceiveMessageWaitTimeSeconds": attrs["ReceiveMessageWaitTimeSeconds"],
					"VisibilityTimeout":             attrs["VisibilityTimeout"],
					"RedrivePolicy":                 helpers.FormatJSONIndentOrRaw(attrs["RedrivePolicy"]),
					// Convert timestamps (usually epoch seconds) into *time.Time for readability.
					// If parsing fails (non-numeric), attempt RFC3339 parse. If both fail, keep raw string.
					"CreatedTimestamp":      helpers.ParseTimestamp(attrs["CreatedTimestamp"]),
					"LastModifiedTimestamp": helpers.ParseTimestamp(attrs["LastModifiedTimestamp"]),
				},
			}))
		}
	}

	return resources, nil
}
