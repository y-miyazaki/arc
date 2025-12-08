// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// CloudWatchLogsCollector collects CloudWatch Logs resources.
// It uses dependency injection to manage CloudWatch Logs clients for multiple regions.
type CloudWatchLogsCollector struct {
	clients      map[string]*cloudwatchlogs.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewCloudWatchLogsCollector creates a new CloudWatch Logs collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create CloudWatch Logs clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *CloudWatchLogsCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewCloudWatchLogsCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*CloudWatchLogsCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *cloudwatchlogs.Client {
		return cloudwatchlogs.NewFromConfig(*c, func(o *cloudwatchlogs.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch Logs clients: %w", err)
	}

	return &CloudWatchLogsCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*CloudWatchLogsCollector) Name() string {
	return "cloudwatch_logs"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*CloudWatchLogsCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*CloudWatchLogsCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "RetentionInDays", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RetentionInDays") }},
		{Header: "StoredBytes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "StoredBytes") }},
		{Header: "MetricFilterCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MetricFilterCount") }},
		{Header: "SubscriptionFilterCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SubscriptionFilterCount") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "CreationTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationTime") }},
	}
}

// Collect collects CloudWatch Logs resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *CloudWatchLogsCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Get all KMS keys to resolve names efficiently
	kmsMap, err := c.nameResolver.GetAllKMSKeys(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}

	paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc, &cloudwatchlogs.DescribeLogGroupsInput{})
	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to describe log groups: %w", pageErr)
		}

		for i := range page.LogGroups {
			lg := &page.LogGroups[i]
			var creationTime string
			if lg.CreationTime != nil {
				t := time.Unix(*lg.CreationTime/1000, 0).UTC() //nolint:mnd
				creationTime = t.Format(time.RFC3339)
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "cloudwatch",
				SubCategory: "LogGroup",
				Name:        lg.LogGroupName,
				Region:      region,
				ARN:         lg.Arn,
				RawData: map[string]any{
					"RetentionInDays":   lg.RetentionInDays,
					"StoredBytes":       lg.StoredBytes,
					"MetricFilterCount": lg.MetricFilterCount,
					// "SubscriptionFilterCount": 0, // Not available in v2 SDK LogGroup struct
					"KmsKey":       helpers.ResolveNameFromMap(lg.KmsKeyId, kmsMap),
					"CreationTime": creationTime,
				},
			}))
		}
	}

	return resources, nil
}
