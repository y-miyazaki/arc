package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// IAMRoleCollector collects IAM Roles.
type IAMRoleCollector struct{}

// Name returns the resource name of the collector.
func (*IAMRoleCollector) Name() string {
	return "iam_role"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*IAMRoleCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*IAMRoleCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Path", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Path") }},
		{Header: "AttachedPolicies", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttachedPolicies") }},
		{Header: "PermissionsBoundary", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PermissionsBoundary") }},
		{Header: "CreateDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreateDate") }},
		{Header: "LastUsedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastUsedDate") }},
	}
}

// Collect collects IAM Roles from AWS
// IAM is a global service, so this only runs in us-east-1 region
func (*IAMRoleCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// IAM is a global service, only process from us-east-1
	if region != "us-east-1" {
		return nil, nil
	}

	svc := iam.NewFromConfig(*cfg)
	var resources []Resource

	// List Roles - collect all IAM roles
	rolePaginator := iam.NewListRolesPaginator(svc, &iam.ListRolesInput{})
	for rolePaginator.HasMorePages() {
		page, err := rolePaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list roles: %w", err)
		}
		for i := range page.Roles {
			role := &page.Roles[i]

			// Get attached policies for the role
			var attachedPolicies []string
			policyPaginator := iam.NewListAttachedRolePoliciesPaginator(svc, &iam.ListAttachedRolePoliciesInput{
				RoleName: role.RoleName,
			})
			for policyPaginator.HasMorePages() {
				policyPage, policyErr := policyPaginator.NextPage(ctx)
				if policyErr != nil {
					return nil, fmt.Errorf("failed to list attached policies for role %s: %w", *role.RoleName, policyErr)
				}
				for _, policy := range policyPage.AttachedPolicies {
					attachedPolicies = append(attachedPolicies, *policy.PolicyArn)
				}
			}
			var permissionsBoundary string
			if role.PermissionsBoundary != nil {
				permissionsBoundary = *role.PermissionsBoundary.PermissionsBoundaryArn
			}
			var lastUsedDate *time.Time
			if role.RoleLastUsed != nil {
				lastUsedDate = role.RoleLastUsed.LastUsedDate
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "iam_role_policy",
				SubCategory: "Role",
				Name:        role.RoleName,
				Region:      "Global",
				ARN:         role.Arn,
				RawData: map[string]any{
					"Path":                role.Path,
					"AttachedPolicies":    attachedPolicies,
					"PermissionsBoundary": permissionsBoundary,
					"CreateDate":          role.CreateDate,
					"LastUsedDate":        lastUsedDate,
				},
			}))
		}
	}

	return resources, nil
}
