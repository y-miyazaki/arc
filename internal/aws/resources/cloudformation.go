// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// CloudFormationCollector collects CloudFormation stacks and stack sets.
// It retrieves details such as outputs, parameters, and drift status.
type CloudFormationCollector struct{}

// Name returns the collector name.
func (*CloudFormationCollector) Name() string {
	return "cloudformation"
}

// ShouldSort returns true.
func (*CloudFormationCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for CloudFormation.
func (*CloudFormationCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Outputs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Outputs") }},
		{Header: "Parameters", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Parameters") }},
		{Header: "Resources", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Resources") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "DriftStatus", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DriftStatus") }},
		{Header: "CreatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedDate") }},
		{Header: "UpdatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "UpdatedDate") }},
	}
}

// Collect collects CloudFormation stacks and stack sets from the specified region.
func (*CloudFormationCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := cloudformation.NewFromConfig(*cfg, func(o *cloudformation.Options) {
		o.Region = region
	})

	var resources []Resource

	// List Stacks
	stackStatusFilter := []types.StackStatus{
		types.StackStatusCreateComplete,
		types.StackStatusUpdateComplete,
		types.StackStatusRollbackComplete,
	}

	stackPaginator := cloudformation.NewListStacksPaginator(svc, &cloudformation.ListStacksInput{
		StackStatusFilter: stackStatusFilter,
	})
	for stackPaginator.HasMorePages() {
		page, err := stackPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list stacks: %w", err)
		}

		for i := range page.StackSummaries {
			stackSummary := &page.StackSummaries[i]
			// Get Stack Details for Outputs and Parameters
			describeOut, descErr := svc.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
				StackName: stackSummary.StackName,
			})
			if descErr != nil || len(describeOut.Stacks) == 0 {
				continue
			}
			stack := describeOut.Stacks[0]

			// Format Outputs
			var outputs []string
			for j := range stack.Outputs {
				o := &stack.Outputs[j]
				key := aws.ToString(o.OutputKey)
				val := aws.ToString(o.OutputValue)
				outputs = append(outputs, fmt.Sprintf("%s=%s", key, val))
			}

			// Format Parameters
			var params []string
			for j := range stack.Parameters {
				p := &stack.Parameters[j]
				key := aws.ToString(p.ParameterKey)
				val := aws.ToString(p.ParameterValue)
				params = append(params, fmt.Sprintf("%s=%s", key, val))
			}

			// List Stack Resources (Summaries)
			var stackResources []string
			resPaginator := cloudformation.NewListStackResourcesPaginator(svc, &cloudformation.ListStackResourcesInput{
				StackName: stackSummary.StackName,
			})
			for resPaginator.HasMorePages() {
				resPage, resErr := resPaginator.NextPage(ctx)
				if resErr != nil {
					break
				}
				for j := range resPage.StackResourceSummaries {
					r := &resPage.StackResourceSummaries[j]
					stackResources = append(stackResources, aws.ToString(r.LogicalResourceId))
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "cloudformation",
				SubCategory: "Stack",
				Name:        stack.StackName,
				Region:      region,
				ARN:         stack.StackId,
				RawData: map[string]any{
					"Description": stack.Description,
					"Type":        "Stack",
					"Outputs":     outputs,
					"Parameters":  params,
					"Resources":   stackResources,
					"Status":      stack.StackStatus,
					"DriftStatus": stack.DriftInformation.StackDriftStatus,
					"CreatedDate": stack.CreationTime,
					"UpdatedDate": stack.LastUpdatedTime,
				},
			}))
		}
	}

	// List StackSets
	ssPaginator := cloudformation.NewListStackSetsPaginator(svc, &cloudformation.ListStackSetsInput{
		Status: types.StackSetStatusActive,
	})
	for ssPaginator.HasMorePages() {
		page, err := ssPaginator.NextPage(ctx)
		if err != nil {
			break
		}

		for i := range page.Summaries {
			ssSummary := &page.Summaries[i]
			// Get StackSet Details
			ssOut, ssErr := svc.DescribeStackSet(ctx, &cloudformation.DescribeStackSetInput{
				StackSetName: ssSummary.StackSetName,
			})
			if ssErr != nil {
				continue
			}
			ss := ssOut.StackSet

			// Format Parameters
			var params []string
			for j := range ss.Parameters {
				p := &ss.Parameters[j]
				key := aws.ToString(p.ParameterKey)
				val := aws.ToString(p.ParameterValue)
				params = append(params, fmt.Sprintf("%s=%s", key, val))
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "cloudformation",
				SubCategory: "StackSet",
				Name:        ss.StackSetName,
				Region:      region,
				ARN:         ss.StackSetARN,
				RawData: map[string]any{
					"Description": ss.Description,
					"Type":        "StackSet",
					"Parameters":  params,
					"Status":      ss.Status,
				},
			}))
		}
	}

	return resources, nil //nolint:nilerr
}
