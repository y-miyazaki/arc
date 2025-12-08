package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

const formatInt = "%d"

// EventBridgeCollector collects EventBridge Rules and Schedules.
// It retrieves details such as schedule expressions, targets, and retry policies.
// It supports both standard EventBridge Rules and the newer EventBridge Scheduler.
// The collector gathers target information including Role ARNs and retry configurations.
// The collector handles pagination manually for ListRules where paginators are not available.
type EventBridgeCollector struct{}

// Name returns the collector name.
func (*EventBridgeCollector) Name() string {
	return "eventbridge"
}

// ShouldSort returns true.
func (*EventBridgeCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for EventBridge.
func (*EventBridgeCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "ScheduleExpression", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ScheduleExpression") }},
		{Header: "Target", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Target") }},
		{Header: "RetryMaxAttempts", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RetryMaxAttempts") }},
		{Header: "RetryMaxEventAgeSeconds", Value: func(r Resource) string {
			return helpers.GetMapValue(r.RawData, "RetryMaxEventAgeSeconds")
		}},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

// Collect collects EventBridge resources from the specified region.
func (*EventBridgeCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	ebSvc := eventbridge.NewFromConfig(*cfg, func(o *eventbridge.Options) {
		o.Region = region
	})
	schSvc := scheduler.NewFromConfig(*cfg, func(o *scheduler.Options) {
		o.Region = region
	})

	var resources []Resource

	// EventBridge Rules
	var nextToken *string
	for {
		page, err := ebSvc.ListRules(ctx, &eventbridge.ListRulesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list rules: %w", err)
		}

		for i := range page.Rules {
			rule := &page.Rules[i]

			// Get Targets
			var roleARN, targetARN, maxAttempts, maxAge string
			targetsOut, targetErr := ebSvc.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
				Rule: rule.Name,
			})
			if targetErr == nil && len(targetsOut.Targets) > 0 {
				t := targetsOut.Targets[0]
				roleARN = aws.ToString(t.RoleArn)
				targetARN = aws.ToString(t.Arn)
				if t.RetryPolicy != nil {
					if t.RetryPolicy.MaximumRetryAttempts != nil {
						maxAttempts = fmt.Sprintf(formatInt, *t.RetryPolicy.MaximumRetryAttempts)
					}
					if t.RetryPolicy.MaximumEventAgeInSeconds != nil {
						maxAge = fmt.Sprintf(formatInt, *t.RetryPolicy.MaximumEventAgeInSeconds)
					}
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "eventbridge",
				SubCategory: "Rule",
				Name:        rule.Name,
				Region:      region,
				ARN:         rule.Arn,
				RawData: map[string]any{
					"Description":             rule.Description,
					"RoleARN":                 roleARN,
					"ScheduleExpression":      rule.ScheduleExpression,
					"Target":                  targetARN,
					"RetryMaxAttempts":        maxAttempts,
					"RetryMaxEventAgeSeconds": maxAge,
					"State":                   rule.State,
				},
			}))
		}

		if page.NextToken == nil {
			break
		}
		nextToken = page.NextToken
	}

	// EventBridge Schedules
	schPaginator := scheduler.NewListSchedulesPaginator(schSvc, &scheduler.ListSchedulesInput{})
	for schPaginator.HasMorePages() {
		page, err := schPaginator.NextPage(ctx)
		if err != nil {
			break
		}

		for i := range page.Schedules {
			schSummary := &page.Schedules[i]

			// Get Schedule Details
			sch, schErr := schSvc.GetSchedule(ctx, &scheduler.GetScheduleInput{
				Name: schSummary.Name,
			})
			if schErr != nil {
				continue
			}

			var roleARN, targetARN, maxAttempts, maxAge string
			if sch.Target != nil {
				roleARN = aws.ToString(sch.Target.RoleArn)
				targetARN = aws.ToString(sch.Target.Arn)
				if sch.Target.RetryPolicy != nil {
					if sch.Target.RetryPolicy.MaximumRetryAttempts != nil {
						maxAttempts = fmt.Sprintf(formatInt, *sch.Target.RetryPolicy.MaximumRetryAttempts)
					}
					if sch.Target.RetryPolicy.MaximumEventAgeInSeconds != nil {
						maxAge = fmt.Sprintf(formatInt, *sch.Target.RetryPolicy.MaximumEventAgeInSeconds)
					}
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "eventbridge",
				SubCategory: "Scheduler",
				Name:        sch.Name,
				Region:      region,
				ARN:         sch.Arn,
				RawData: map[string]any{
					"Description":             sch.Description,
					"RoleARN":                 roleARN,
					"ScheduleExpression":      sch.ScheduleExpression,
					"Target":                  targetARN,
					"RetryMaxAttempts":        maxAttempts,
					"RetryMaxEventAgeSeconds": maxAge,
					"State":                   sch.State,
				},
			}))
		}
	}

	return resources, nil //nolint:nilerr
}
