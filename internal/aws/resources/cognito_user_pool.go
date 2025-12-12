// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// CognitoUserPoolCollector collects Cognito User Pools, Groups, and Users.
// It uses dependency injection to manage Cognito Identity Provider clients for multiple regions.
type CognitoUserPoolCollector struct {
	clients      map[string]*cognitoidentityprovider.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewCognitoUserPoolCollector creates a new Cognito User Pool collector with clients for the specified regions.
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Cognito Identity Provider clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *CognitoUserPoolCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewCognitoUserPoolCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*CognitoUserPoolCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *cognitoidentityprovider.Client {
		return cognitoidentityprovider.NewFromConfig(*c, func(o *cognitoidentityprovider.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Cognito Identity Provider clients: %w", err)
	}

	return &CognitoUserPoolCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*CognitoUserPoolCollector) Name() string {
	return "cognito_user_pool"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*CognitoUserPoolCollector) ShouldSort() bool {
	return false
}

// GetColumns returns the CSV columns for the collector.
func (*CognitoUserPoolCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "SubCategory2", Value: func(r Resource) string { return r.SubCategory2 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "MfaConfiguration", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MfaConfiguration") }},
		{Header: "AliasAttributes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AliasAttributes") }},
		{Header: "UsernameAttributes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "UsernameAttributes") }},
		{Header: "AutoVerifiedAttributes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AutoVerifiedAttributes") }},
		{Header: "PasswordPolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PasswordPolicy") }},
		{Header: "LambdaConfig", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LambdaConfig") }},
		{Header: "Precedence", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Precedence") }},
		{Header: "RoleArn", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleArn") }},
		{Header: "AttachedUsers", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttachedUsers") }},
		{Header: "Groups", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Groups") }},
		{Header: "Attributes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Attributes") }},
		{Header: "CreationDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationDate") }},
		{Header: "LastModifiedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastModifiedDate") }},
	}
}

// Collect collects Cognito User Pool resources for the specified region.
// The collector must have been initialized with clients for this region.
func (c *CognitoUserPoolCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	client, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	resources, err := collectUserPools(ctx, region, client)
	if err != nil {
		return nil, fmt.Errorf("failed to collect user pools: %w", err)
	}
	return resources, nil
}

// collectUserPools lists user pools, groups, and users and returns resources.
func collectUserPools(ctx context.Context, region string, idpSvc *cognitoidentityprovider.Client) ([]Resource, error) {
	var resources []Resource
	paginator := cognitoidentityprovider.NewListUserPoolsPaginator(idpSvc, &cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: aws.Int32(maxResults),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list user pools: %w", err)
		}

		for i := range page.UserPools {
			pool := &page.UserPools[i]

			// Collect authentication / alias configuration from the pool description where available.
			var (
				mfaConfig      string
				aliasAttrs     any
				usernameAttrs  any
				autoVerified   any
				passwordPolicy []string
				lambdaConfig   []string
				poolARN        *string
			)
			if descOut, derr := idpSvc.DescribeUserPool(ctx, &cognitoidentityprovider.DescribeUserPoolInput{UserPoolId: pool.Id}); derr == nil && descOut != nil && descOut.UserPool != nil {
				up := descOut.UserPool
				mfaConfig = fmt.Sprint(up.MfaConfiguration)
				aliasAttrs = up.AliasAttributes
				usernameAttrs = up.UsernameAttributes
				autoVerified = up.AutoVerifiedAttributes
				poolARN = up.Arn
				if up.Policies != nil && up.Policies.PasswordPolicy != nil { // pragma: allowlist secret
					passwordPolicy = helpers.StructToKeyValue(up.Policies.PasswordPolicy)
				}
				lambdaConfig = helpers.StructToKeyValue(up.LambdaConfig)
			}
			resources = append(resources, NewResource(&ResourceInput{
				Category:     "cognito",
				SubCategory1: "UserPool",
				Name:         pool.Name,
				Region:       region,
				ARN:          poolARN,
				RawData: map[string]any{
					"ID":                     pool.Id,
					"MfaConfiguration":       mfaConfig,
					"AliasAttributes":        aliasAttrs,
					"UsernameAttributes":     usernameAttrs,
					"AutoVerifiedAttributes": autoVerified,
					"PasswordPolicy":         passwordPolicy,
					"LambdaConfig":           lambdaConfig,
					"CreationDate":           pool.CreationDate,
					"LastModifiedDate":       pool.LastModifiedDate,
				},
			}))

			// --- Groups and Users for this User Pool ---
			// Collect groups first and build a mapping of username -> []groups
			groupPaginator := cognitoidentityprovider.NewListGroupsPaginator(idpSvc, &cognitoidentityprovider.ListGroupsInput{
				UserPoolId: pool.Id,
				Limit:      aws.Int32(maxResults),
			})

			userGroups := make(map[string][]string)
			var groupResources []Resource

			for groupPaginator.HasMorePages() {
				gpage, gerr := groupPaginator.NextPage(ctx)
				if gerr != nil {
					return nil, fmt.Errorf("failed to list groups for user pool %s: %w", helpers.StringValue(pool.Id), gerr)
				}
				for j := range gpage.Groups {
					grp := &gpage.Groups[j]

					// For each group, collect member usernames
					var attachedUsers []string
					usersInGroupPaginator := cognitoidentityprovider.NewListUsersInGroupPaginator(idpSvc, &cognitoidentityprovider.ListUsersInGroupInput{
						UserPoolId: pool.Id,
						GroupName:  grp.GroupName,
						Limit:      aws.Int32(maxResults),
					})
					for usersInGroupPaginator.HasMorePages() {
						upage, uerr := usersInGroupPaginator.NextPage(ctx)
						if uerr != nil {
							return nil, fmt.Errorf("failed to list users in group %s for pool %s: %w", helpers.StringValue(grp.GroupName), helpers.StringValue(pool.Id), uerr)
						}
						for k := range upage.Users {
							u := &upage.Users[k]
							username := helpers.StringValue(u.Username)
							attachedUsers = append(attachedUsers, username)
							userGroups[username] = append(userGroups[username], helpers.StringValue(grp.GroupName))
						}
					}

					// Build group resource (do not emit user resources here)
					groupResources = append(groupResources, NewResource(&ResourceInput{
						Category:     "cognito",
						SubCategory1: "",
						SubCategory2: "Group",
						Name:         grp.GroupName,
						Region:       region,
						RawData: map[string]any{
							"Description":      grp.Description,
							"Precedence":       grp.Precedence,
							"RoleArn":          grp.RoleArn,
							"AttachedUsers":    attachedUsers,
							"CreationDate":     grp.CreationDate,
							"LastModifiedDate": grp.LastModifiedDate,
						},
					}))
				}
			}

			// Emit collected group resources
			for i := range groupResources {
				resources = append(resources, groupResources[i])
			}

			// Now list all users and emit them as SubCategory2 "User" with Groups in RawData
			usersPaginator := cognitoidentityprovider.NewListUsersPaginator(idpSvc, &cognitoidentityprovider.ListUsersInput{
				UserPoolId: pool.Id,
				Limit:      aws.Int32(maxResults),
			})
			for usersPaginator.HasMorePages() {
				upage, uerr := usersPaginator.NextPage(ctx)
				if uerr != nil {
					return nil, fmt.Errorf("failed to list users for pool %s: %w", helpers.StringValue(pool.Id), uerr)
				}
				for k := range upage.Users {
					u := &upage.Users[k]
					username := helpers.StringValue(u.Username)

					// Build attribute list for the user as key=value strings to match other collectors' slice style
					var attrPairs []string
					for _, a := range u.Attributes {
						attrPairs = append(attrPairs, fmt.Sprintf("%s=%s", *a.Name, *a.Value))
					}
					attrPairs = append(attrPairs, fmt.Sprintf("Enabled=%s", helpers.StringValue(u.Enabled)))
					for i := range u.MFAOptions {
						attrPairs = append(attrPairs, fmt.Sprintf("MFAOptions %s=%s", *u.MFAOptions[i].AttributeName, helpers.StringValue(u.MFAOptions[i].DeliveryMedium)))
					}
					attrPairs = append(attrPairs, fmt.Sprintf("UserStatus=%s", helpers.StringValue(u.UserStatus)))

					var groupsSlice []string
					if groups, exists := userGroups[username]; exists {
						groupsSlice = groups
					}

					resources = append(resources, NewResource(&ResourceInput{
						Category:     "cognito",
						SubCategory1: "",
						SubCategory2: "User",
						Name:         username,
						Region:       region,
						RawData: map[string]any{
							"Groups":           groupsSlice,
							"Attributes":       attrPairs,
							"CreationDate":     u.UserCreateDate,
							"LastModifiedDate": u.UserLastModifiedDate,
						},
					}))
				}
			}
		}
	}
	return resources, nil
}
