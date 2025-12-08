// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

const formatInt = "%d"

// EventBridgeCollector collects EventBridge Rules and Schedules.
// It uses dependency injection to manage EventBridge and Scheduler clients for multiple regions.
// It retrieves details such as schedule expressions, targets, and retry policies.
// It supports both standard EventBridge Rules and the newer EventBridge Scheduler.
// The collector gathers target information including Role ARNs and retry configurations.
// The collector handles pagination manually for ListRules where paginators are not available.
type EventBridgeCollector struct {
	ebClients    map[string]*eventbridge.Client
	schClients   map[string]*scheduler.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewEventBridgeCollector creates a new EventBridge collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create EventBridge clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *EventBridgeCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewEventBridgeCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*EventBridgeCollector, error) {
	ebClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *eventbridge.Client {
		return eventbridge.NewFromConfig(*c, func(o *eventbridge.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create EventBridge clients: %w", err)
	}

	schClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *scheduler.Client {
		return scheduler.NewFromConfig(*c, func(o *scheduler.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Scheduler clients: %w", err)
	}

	return &EventBridgeCollector{
		ebClients:    ebClients,
		schClients:   schClients,
		nameResolver: nameResolver,
	}, nil
}

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

// Collect collects EventBridge resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *EventBridgeCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	ebSvc, ok := c.ebClients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	schSvc, ok := c.schClients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s (Scheduler)", ErrNoClientForRegion, region)
	}

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
