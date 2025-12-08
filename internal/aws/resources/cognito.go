package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

const maxResults = 60

// CognitoCollector collects Cognito User Pools and Identity Pools.
// It retrieves status and creation dates for User Pools.
// It also collects configuration for Identity Pools, such as unauthenticated access.
// The collector paginates through both User Pools and Identity Pools to ensure complete coverage.
type CognitoCollector struct{}

// Name returns the collector name.
func (*CognitoCollector) Name() string {
	return "cognito"
}

// ShouldSort returns true.
func (*CognitoCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for Cognito.
func (*CognitoCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID
		{Header: "AllowUnauthenticated", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AllowUnauthenticated") }},
		{Header: "CreationDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationDate") }},
		{Header: "LastModifiedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastModifiedDate") }},
	}
}

// Collect collects Cognito resources from the specified region.
func (*CognitoCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	idpSvc := cognitoidentityprovider.NewFromConfig(*cfg, func(o *cognitoidentityprovider.Options) {
		o.Region = region
	})
	identitySvc := cognitoidentity.NewFromConfig(*cfg, func(o *cognitoidentity.Options) {
		o.Region = region
	})

	var resources []Resource

	// User Pools
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
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "cognito",
				SubCategory: "UserPool",
				Name:        pool.Name,
				Region:      region,
				ARN:         pool.Id,
				RawData: map[string]any{
					"CreationDate":     pool.CreationDate,
					"LastModifiedDate": pool.LastModifiedDate,
				},
			}))
		}
	}

	// Identity Pools
	var nextToken *string
	for {
		out, err := identitySvc.ListIdentityPools(ctx, &cognitoidentity.ListIdentityPoolsInput{
			MaxResults: aws.Int32(maxResults),
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list identity pools: %w", err)
		}

		for i := range out.IdentityPools {
			pool := &out.IdentityPools[i]

			var allowUnauth bool
			desc, descErr := identitySvc.DescribeIdentityPool(ctx, &cognitoidentity.DescribeIdentityPoolInput{
				IdentityPoolId: pool.IdentityPoolId,
			})
			if descErr == nil {
				allowUnauth = desc.AllowUnauthenticatedIdentities
			}
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "cognito",
				SubCategory: "IdentityPool",
				Name:        pool.IdentityPoolName,
				Region:      region,
				ARN:         pool.IdentityPoolId,
				RawData: map[string]any{
					"AllowUnauthenticated": allowUnauth,
				},
			}))
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}

	return resources, nil //nolint:nilerr
}
