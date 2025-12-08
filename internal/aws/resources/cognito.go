// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

const maxResults = 60

// CognitoCollector collects Cognito User Pools and Identity Pools.
// It uses dependency injection to manage Cognito clients for multiple regions.
type CognitoCollector struct {
	idpClients      map[string]*cognitoidentityprovider.Client
	identityClients map[string]*cognitoidentity.Client
	nameResolver    *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewCognitoCollector creates a new Cognito collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Cognito clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *CognitoCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewCognitoCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*CognitoCollector, error) {
	idpClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *cognitoidentityprovider.Client {
		return cognitoidentityprovider.NewFromConfig(*c, func(o *cognitoidentityprovider.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Cognito Identity Provider clients: %w", err)
	}

	identityClients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *cognitoidentity.Client {
		return cognitoidentity.NewFromConfig(*c, func(o *cognitoidentity.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Cognito Identity clients: %w", err)
	}

	return &CognitoCollector{
		idpClients:      idpClients,
		identityClients: identityClients,
		nameResolver:    nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*CognitoCollector) Name() string {
	return "cognito"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*CognitoCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
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

// Collect collects Cognito resources for the specified region.
// The collector must have been initialized with clients for this region.
func (c *CognitoCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	idpSvc, ok := c.idpClients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	identitySvc, ok := c.identityClients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

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
