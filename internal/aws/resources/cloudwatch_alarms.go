// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// CloudWatchAlarmsCollector collects CloudWatch alarms.
// It collects both Metric Alarms and Composite Alarms.
// It uses dependency injection to manage CloudWatch clients for multiple regions.
type CloudWatchAlarmsCollector struct {
	clients      map[string]*cloudwatch.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewCloudWatchAlarmsCollector creates a new CloudWatch Alarms collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create CloudWatch clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *CloudWatchAlarmsCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewCloudWatchAlarmsCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*CloudWatchAlarmsCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *cloudwatch.Client {
		return cloudwatch.NewFromConfig(*c, func(o *cloudwatch.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudWatch clients: %w", err)
	}

	return &CloudWatchAlarmsCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the collector name.
func (*CloudWatchAlarmsCollector) Name() string {
	return "cloudwatch_alarms"
}

// ShouldSort returns true.
func (*CloudWatchAlarmsCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for CloudWatch alarms.
func (*CloudWatchAlarmsCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "MetricName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MetricName") }},
		{Header: "Namespace", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Namespace") }},
		{Header: "Statistic", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Statistic") }},
		{Header: "Threshold", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Threshold") }},
		{Header: "ComparisonOperator", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ComparisonOperator") }},
		{Header: "EvaluationPeriods", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EvaluationPeriods") }},
		{Header: "Period", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Period") }},
		{Header: "TreatMissingData", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TreatMissingData") }},
	}
}

// Collect collects CloudWatch alarms for the specified region.
// The collector must have been initialized with a client for this region.
func (c *CloudWatchAlarmsCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	paginator := cloudwatch.NewDescribeAlarmsPaginator(svc, &cloudwatch.DescribeAlarmsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe alarms: %w", err)
		}

		for i := range page.MetricAlarms {
			alarm := &page.MetricAlarms[i]
			// Dereference Threshold pointer to get the actual value
			resources = append(resources, NewResource(&ResourceInput{
				Category:     "cloudwatch",
				SubCategory1: "Alarm",
				Name:         alarm.AlarmName,
				Region:       region,
				ARN:          alarm.AlarmArn,
				RawData: map[string]any{
					"MetricName":         alarm.MetricName,
					"Namespace":          alarm.Namespace,
					"Statistic":          alarm.Statistic,
					"Threshold":          alarm.Threshold,
					"ComparisonOperator": alarm.ComparisonOperator,
					"EvaluationPeriods":  alarm.EvaluationPeriods,
					"Period":             alarm.Period,
					"TreatMissingData":   alarm.TreatMissingData,
				},
			}))
		}

		for i := range page.CompositeAlarms {
			alarm := &page.CompositeAlarms[i]
			resources = append(resources, NewResource(&ResourceInput{
				Category:     "cloudwatch",
				SubCategory1: "Alarm",
				Name:         alarm.AlarmName,
				Region:       region,
				ARN:          alarm.AlarmArn,
				RawData: map[string]any{
					"MetricName": "Composite",
					"Namespace":  "Composite",
				},
			}))
		}
	}

	return resources, nil
}
