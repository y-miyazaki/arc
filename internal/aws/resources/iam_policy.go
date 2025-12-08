package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// IAMPolicyCollector collects IAM Policies.
type IAMPolicyCollector struct{}

// Name returns the resource name of the collector.
func (*IAMPolicyCollector) Name() string {
	return "iam_policy"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*IAMPolicyCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*IAMPolicyCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "Scope", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Scope") }},
		{Header: "Path", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Path") }},
		{Header: "CreateDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreateDate") }},
	}
}

// Collect collects IAM Policies from AWS
// IAM is a global service, so this only runs in us-east-1 region
func (*IAMPolicyCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// IAM is a global service, only process from us-east-1
	if region != "us-east-1" {
		return nil, nil
	}

	svc := iam.NewFromConfig(*cfg)
	var resources []Resource

	// List Policies (Local scope only) - collect customer-managed policies
	policyPaginator := iam.NewListPoliciesPaginator(svc, &iam.ListPoliciesInput{
		Scope: types.PolicyScopeTypeLocal,
	})
	for policyPaginator.HasMorePages() {
		page, err := policyPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list policies: %w", err)
		}
		for i := range page.Policies {
			policy := &page.Policies[i]

			// Get detailed policy information including description
			var description string
			if policy.Arn != nil {
				getPolicyInput := &iam.GetPolicyInput{
					PolicyArn: policy.Arn,
				}
				getPolicyOutput, getErr := svc.GetPolicy(ctx, getPolicyInput)
				if getErr == nil && getPolicyOutput.Policy != nil && getPolicyOutput.Policy.Description != nil {
					description = *getPolicyOutput.Policy.Description
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "iam_policy",
				SubCategory: "Policy",
				Name:        policy.PolicyName,
				Region:      "Global",
				ARN:         policy.Arn,
				RawData: map[string]any{
					"Description": description,
					"Scope":       string(types.PolicyScopeTypeLocal),
					"Path":        policy.Path,
					"CreateDate":  policy.CreateDate,
				},
			}))
		}
	}

	return resources, nil
}
