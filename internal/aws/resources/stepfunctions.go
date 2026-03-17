// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// StepFunctionsCollector collects AWS Step Functions state machines and activities.
// It uses dependency injection to manage Step Functions clients for multiple regions.
type StepFunctionsCollector struct {
	clients      map[string]*sfn.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewStepFunctionsCollector creates a new Step Functions collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Step Functions clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *StepFunctionsCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewStepFunctionsCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*StepFunctionsCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *sfn.Client {
		return sfn.NewFromConfig(*c, func(o *sfn.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Step Functions clients: %w", err)
	}

	return &StepFunctionsCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*StepFunctionsCollector) Name() string {
	return "stepfunctions"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*StepFunctionsCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*StepFunctionsCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Comment", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Comment") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "LoggingLevel", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LoggingLevel") }},
		{Header: "LoggingIncludeExecutionData", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LoggingIncludeExecutionData") }},
		{Header: "LogDestination", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LogDestination") }},
		{Header: "TracingEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TracingEnabled") }},
		{Header: "EncryptionType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EncryptionType") }},
		{Header: "KMSKeyID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KMSKeyID") }},
		{Header: "KMSDataKeyReusePeriodSeconds", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KMSDataKeyReusePeriodSeconds") }},
		{Header: "Definition", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Definition") }},
		{Header: "RevisionID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RevisionID") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "CreatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedDate") }},
	}
}

// Collect collects Step Functions resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *StepFunctionsCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	resources := make([]Resource, 0)

	stateMachinePaginator := sfn.NewListStateMachinesPaginator(svc, &sfn.ListStateMachinesInput{})
	for stateMachinePaginator.HasMorePages() {
		page, err := stateMachinePaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list state machines: %w", err)
		}

		for i := range page.StateMachines {
			stateMachine := &page.StateMachines[i]
			description, descErr := svc.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
				StateMachineArn: stateMachine.StateMachineArn,
			})
			if descErr != nil {
				continue
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "stepfunctions",
				SubCategory1: "StateMachine",
				Name:         stateMachine.Name,
				Region:       region,
				ARN:          stateMachine.StateMachineArn,
				RawData: map[string]any{
					"Type":                         string(description.Type),
					"Status":                       string(description.Status),
					"RoleARN":                      description.RoleArn,
					"LoggingLevel":                 getLoggingLevel(description.LoggingConfiguration),
					"LoggingIncludeExecutionData":  getLoggingIncludeExecutionData(description.LoggingConfiguration),
					"LogDestination":               getLogDestination(description.LoggingConfiguration),
					"TracingEnabled":               getTracingEnabled(description.TracingConfiguration),
					"EncryptionType":               getEncryptionType(description.EncryptionConfiguration),
					"KMSKeyID":                     getEncryptionKeyID(description.EncryptionConfiguration),
					"KMSDataKeyReusePeriodSeconds": getEncryptionKeyReusePeriod(description.EncryptionConfiguration),
					"Definition":                   description.Definition,
					"RevisionID":                   description.RevisionId,
					"CreatedDate":                  description.CreationDate,
					"Comment":                      getDefinitionComment(description.Definition),
				},
			}))
		}
	}

	activityPaginator := sfn.NewListActivitiesPaginator(svc, &sfn.ListActivitiesInput{})
	for activityPaginator.HasMorePages() {
		page, err := activityPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list activities: %w", err)
		}

		for i := range page.Activities {
			activity := &page.Activities[i]
			description, descErr := svc.DescribeActivity(ctx, &sfn.DescribeActivityInput{
				ActivityArn: activity.ActivityArn,
			})
			if descErr != nil {
				continue
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "stepfunctions",
				SubCategory1: "Activity",
				Name:         activity.Name,
				Region:       region,
				ARN:          activity.ActivityArn,
				RawData: map[string]any{
					"Type":                         "Activity",
					"EncryptionType":               getEncryptionType(description.EncryptionConfiguration),
					"KMSKeyID":                     getEncryptionKeyID(description.EncryptionConfiguration),
					"KMSDataKeyReusePeriodSeconds": getEncryptionKeyReusePeriod(description.EncryptionConfiguration),
					"CreatedDate":                  description.CreationDate,
				},
			}))
		}
	}

	return resources, nil
}

func getDefinitionComment(definition *string) string {
	if definition == nil {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(*definition), &data); err != nil {
		return ""
	}
	comment, ok := data["Comment"].(string)
	if !ok {
		return ""
	}
	return comment
}

func getEncryptionKeyID(cfg *sfntypes.EncryptionConfiguration) string {
	if cfg == nil {
		return ""
	}
	return aws.ToString(cfg.KmsKeyId)
}

func getEncryptionKeyReusePeriod(cfg *sfntypes.EncryptionConfiguration) any {
	if cfg == nil || cfg.KmsDataKeyReusePeriodSeconds == nil {
		return ""
	}
	return *cfg.KmsDataKeyReusePeriodSeconds
}

func getEncryptionType(cfg *sfntypes.EncryptionConfiguration) string {
	if cfg == nil {
		return ""
	}
	return string(cfg.Type)
}

func getLogDestination(cfg *sfntypes.LoggingConfiguration) string {
	if cfg == nil || len(cfg.Destinations) == 0 || cfg.Destinations[0].CloudWatchLogsLogGroup == nil {
		return ""
	}
	return aws.ToString(cfg.Destinations[0].CloudWatchLogsLogGroup.LogGroupArn)
}

func getLoggingIncludeExecutionData(cfg *sfntypes.LoggingConfiguration) any {
	if cfg == nil {
		return ""
	}
	return cfg.IncludeExecutionData
}

func getLoggingLevel(cfg *sfntypes.LoggingConfiguration) string {
	if cfg == nil {
		return ""
	}
	return string(cfg.Level)
}

func getTracingEnabled(cfg *sfntypes.TracingConfiguration) any {
	if cfg == nil {
		return ""
	}
	return cfg.Enabled
}
