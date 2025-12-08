// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

// CloudWatchAlarmsCollector collects CloudWatch alarms.
// It collects both Metric Alarms and Composite Alarms.
type CloudWatchAlarmsCollector struct{}

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
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
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

// Collect collects CloudWatch alarms from the specified region.
func (*CloudWatchAlarmsCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := cloudwatch.NewFromConfig(*cfg, func(o *cloudwatch.Options) {
		o.Region = region
	})

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
				Category:    "cloudwatch",
				SubCategory: "Alarm",
				Name:        alarm.AlarmName,
				Region:      region,
				ARN:         alarm.AlarmArn,
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
				Category:    "cloudwatch",
				SubCategory: "Alarm",
				Name:        alarm.AlarmName,
				Region:      region,
				ARN:         alarm.AlarmArn,
				RawData: map[string]any{
					"MetricName": "Composite",
					"Namespace":  "Composite",
				},
			}))
		}
	}

	return resources, nil
}
