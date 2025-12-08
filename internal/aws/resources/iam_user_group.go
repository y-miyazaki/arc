// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// IAMUserGroupCollector collects IAM Users and Groups.
// It uses dependency injection to manage IAM clients.
type IAMUserGroupCollector struct {
	client       *iam.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewIAMUserGroupCollector creates a new IAM User/Group collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions (IAM is global, only processes in us-east-1)
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *IAMUserGroupCollector: Initialized collector with IAM client and name resolver
//   - error: Error if client creation fails
func NewIAMUserGroupCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*IAMUserGroupCollector, error) {
	// IAM is a global service, create single client
	_ = regions // unused parameter
	client := iam.NewFromConfig(*cfg)

	return &IAMUserGroupCollector{
		client:       client,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*IAMUserGroupCollector) Name() string {
	return "iam_user_group"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*IAMUserGroupCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*IAMUserGroupCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Path", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Path") }},
		{Header: "PasswordLastUsed", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PasswordLastUsed") }},
		{Header: "CreateDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreateDate") }},
		{Header: "AttachedUsers", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttachedUsers") }},
		{Header: "AttachedPolicies", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttachedPolicies") }},
	}
}

// Collect collects IAM Users and Groups for the specified region.
// IAM is a global service, so this only runs in us-east-1 region.
func (c *IAMUserGroupCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	// IAM is a global service, only process from us-east-1
	if region != "us-east-1" {
		return nil, nil
	}

	var resources []Resource

	// List Users - collect all IAM users
	userPaginator := iam.NewListUsersPaginator(c.client, &iam.ListUsersInput{})
	for userPaginator.HasMorePages() {
		page, err := userPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}
		for i := range page.Users {
			user := &page.Users[i]
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "iam_user_group",
				SubCategory: "User",
				Name:        user.UserName,
				Region:      "Global",
				ARN:         user.Arn,
				RawData: map[string]any{
					"Path":             user.Path,
					"PasswordLastUsed": user.PasswordLastUsed,
					"CreateDate":       user.CreateDate,
				},
			}))
		}
	}

	// List Groups - collect all IAM groups
	groupPaginator := iam.NewListGroupsPaginator(c.client, &iam.ListGroupsInput{})
	for groupPaginator.HasMorePages() {
		page, err := groupPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list groups: %w", err)
		}
		for i := range page.Groups {
			group := &page.Groups[i]

			// Get group details including attached users
			groupDetails, groupErr := c.client.GetGroup(ctx, &iam.GetGroupInput{
				GroupName: group.GroupName,
			})
			if groupErr != nil {
				return nil, fmt.Errorf("failed to get group details for %s: %w", *group.GroupName, groupErr)
			}

			// Collect attached users
			var attachedUsers []string
			for i := range groupDetails.Users {
				attachedUsers = append(attachedUsers, *groupDetails.Users[i].UserName)
			}

			// Get attached policies for the group
			policyPaginator := iam.NewListAttachedGroupPoliciesPaginator(c.client, &iam.ListAttachedGroupPoliciesInput{
				GroupName: group.GroupName,
			})
			var attachedPolicies []string
			for policyPaginator.HasMorePages() {
				policyPage, policyErr := policyPaginator.NextPage(ctx)
				if policyErr != nil {
					return nil, fmt.Errorf("failed to list attached policies for group %s: %w", *group.GroupName, policyErr)
				}
				for _, policy := range policyPage.AttachedPolicies {
					attachedPolicies = append(attachedPolicies, *policy.PolicyName)
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "iam_user_group",
				SubCategory: "Group",
				Name:        group.GroupName,
				Region:      "Global",
				ARN:         group.Arn,
				RawData: map[string]any{
					"Path":             group.Path,
					"CreateDate":       group.CreateDate,
					"AttachedUsers":    attachedUsers,
					"AttachedPolicies": attachedPolicies,
				},
			}))
		}
	}

	return resources, nil
}
