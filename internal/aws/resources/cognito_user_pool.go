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
		{Header: "SubCategory3", Value: func(r Resource) string { return r.SubCategory3 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID
		{Header: "Groups", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Groups") }},
		{Header: "GroupName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "GroupName") }},
		{Header: "Attributes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Attributes") }},
		{Header: "MfaConfiguration", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MfaConfiguration") }},
		{Header: "AliasAttributes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AliasAttributes") }},
		{Header: "UsernameAttributes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "UsernameAttributes") }},
		{Header: "AutoVerifiedAttributes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AutoVerifiedAttributes") }},
		{Header: "PasswordPolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PasswordPolicy") }},
		{Header: "LambdaConfig", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LambdaConfig") }},
		{Header: "RoleArn", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleArn") }},
		{Header: "AttachedUsers", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttachedUsers") }},
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
			)
			if descOut, derr := idpSvc.DescribeUserPool(ctx, &cognitoidentityprovider.DescribeUserPoolInput{UserPoolId: pool.Id}); derr == nil && descOut != nil && descOut.UserPool != nil {
				up := descOut.UserPool
				mfaConfig = fmt.Sprint(up.MfaConfiguration)
				aliasAttrs = up.AliasAttributes
				usernameAttrs = up.UsernameAttributes
				autoVerified = up.AutoVerifiedAttributes

				if up.Policies != nil && up.Policies.PasswordPolicy != nil { // pragma: allowlist secret
					pp := up.Policies.PasswordPolicy
					if pp.MinimumLength != nil {
						passwordPolicy = append(passwordPolicy, fmt.Sprintf("MinimumLength=%d", *pp.MinimumLength))
					}
					if pp.RequireUppercase {
						passwordPolicy = append(passwordPolicy, "RequireUppercase=true")
					}
					if pp.RequireLowercase {
						passwordPolicy = append(passwordPolicy, "RequireLowercase=true")
					}
					if pp.RequireNumbers {
						passwordPolicy = append(passwordPolicy, "RequireNumbers=true")
					}
					if pp.RequireSymbols {
						passwordPolicy = append(passwordPolicy, "RequireSymbols=true")
					}
				}

				if up.LambdaConfig != nil {
					if up.LambdaConfig.PreSignUp != nil {
						lambdaConfig = append(lambdaConfig, fmt.Sprintf("PreSignUp=%s", helpers.StringValue(up.LambdaConfig.PreSignUp)))
					}
					if up.LambdaConfig.PostConfirmation != nil {
						lambdaConfig = append(lambdaConfig, fmt.Sprintf("PostConfirmation=%s", helpers.StringValue(up.LambdaConfig.PostConfirmation)))
					}
					if up.LambdaConfig.CustomMessage != nil {
						lambdaConfig = append(lambdaConfig, fmt.Sprintf("CustomMessage=%s", helpers.StringValue(up.LambdaConfig.CustomMessage)))
					}
					if up.LambdaConfig.PreAuthentication != nil {
						lambdaConfig = append(lambdaConfig, fmt.Sprintf("PreAuthentication=%s", helpers.StringValue(up.LambdaConfig.PreAuthentication)))
					}
					if up.LambdaConfig.PostAuthentication != nil {
						lambdaConfig = append(lambdaConfig, fmt.Sprintf("PostAuthentication=%s", helpers.StringValue(up.LambdaConfig.PostAuthentication)))
					}
					if up.LambdaConfig.DefineAuthChallenge != nil {
						lambdaConfig = append(lambdaConfig, fmt.Sprintf("DefineAuthChallenge=%s", helpers.StringValue(up.LambdaConfig.DefineAuthChallenge)))
					}
					if up.LambdaConfig.CreateAuthChallenge != nil {
						lambdaConfig = append(lambdaConfig, fmt.Sprintf("CreateAuthChallenge=%s", helpers.StringValue(up.LambdaConfig.CreateAuthChallenge)))
					}
					if up.LambdaConfig.VerifyAuthChallengeResponse != nil {
						lambdaConfig = append(lambdaConfig, fmt.Sprintf("VerifyAuthChallengeResponse=%s", helpers.StringValue(up.LambdaConfig.VerifyAuthChallengeResponse)))
					}
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "cognito",
				SubCategory1: "UserPool",
				Name:         pool.Name,
				Region:       region,
				ARN:          pool.Id,
				RawData: map[string]any{
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
						ARN:          grp.GroupName,
						RawData: map[string]any{
							"Description":      grp.Description,
							"Precedence":       grp.Precedence,
							"RoleArn":          grp.RoleArn,
							"LastModifiedDate": grp.LastModifiedDate,
							"AttachedUsers":    attachedUsers,
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
					var groupsSlice []string
					if groups, exists := userGroups[username]; exists {
						groupsSlice = groups
					}

					// Derive verification status from attributes
					verifiedEmail := false
					verifiedPhone := false
					for _, a := range u.Attributes {
						name := helpers.StringValue(a.Name)
						val := helpers.StringValue(a.Value)
						if name == "email_verified" && val == "true" {
							verifiedEmail = true
						}
						if name == "phone_number_verified" && val == "true" {
							verifiedPhone = true
						}
					}

					// Include account status and verification flags in Attributes (combine appends)
					attrPairs = append(attrPairs,
						fmt.Sprintf("AccountEnabled=%t", u.Enabled),
						fmt.Sprintf("UserStatus=%s", helpers.StringValue(u.UserStatus)),
						fmt.Sprintf("VerifiedEmail=%t", verifiedEmail),
						fmt.Sprintf("VerifiedPhone=%t", verifiedPhone),
					)

					// Emit a resource for the user for each group (SubCategory1="", SubCategory2="Group", SubCategory3="User")
					if len(groupsSlice) > 0 {
						for _, grp := range groupsSlice {
							resources = append(resources, NewResource(&ResourceInput{
								Category:     "cognito",
								SubCategory1: "",
								SubCategory2: "Group",
								SubCategory3: "User",
								Name:         username,
								Region:       region,
								ARN:          username,
								RawData: map[string]any{
									"Groups":           groupsSlice,
									"GroupName":        grp,
									"Attributes":       attrPairs,
									"CreationDate":     u.UserCreateDate,
									"LastModifiedDate": u.UserLastModifiedDate,
								},
							}))
						}
					} else {
						// No group: SubCategory1="", SubCategory2="User"
						resources = append(resources, NewResource(&ResourceInput{
							Category:     "cognito",
							SubCategory1: "",
							SubCategory2: "User",
							Name:         username,
							Region:       region,
							ARN:          username,
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
	}
	return resources, nil
}
