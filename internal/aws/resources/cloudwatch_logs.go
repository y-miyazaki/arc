// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// CloudWatchLogsCollector collects CloudWatch Logs resources.
type CloudWatchLogsCollector struct{}

// Name returns the collector name.
func (*CloudWatchLogsCollector) Name() string {
	return "cloudwatch_logs"
}

// ShouldSort returns true.
func (*CloudWatchLogsCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for CloudWatch Logs.
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

// Collect collects CloudWatch Logs resources from the specified region.
func (*CloudWatchLogsCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := cloudwatchlogs.NewFromConfig(*cfg, func(o *cloudwatchlogs.Options) {
		o.Region = region
	})

	var resources []Resource

	// Get all KMS keys to resolve names efficiently
	kmsMap, err := helpers.GetAllKMSKeys(ctx, cfg, region)
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
