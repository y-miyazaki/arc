// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

const maxResults = 60

// CognitoIdentityPoolCollector collects Cognito Identity Pools.
// It uses dependency injection to manage Cognito Identity clients for multiple regions.
type CognitoIdentityPoolCollector struct {
	clients      map[string]*cognitoidentity.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewCognitoIdentityPoolCollector creates a new Cognito Identity Pool collector with clients for the specified regions.
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Cognito Identity clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *CognitoIdentityPoolCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewCognitoIdentityPoolCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*CognitoIdentityPoolCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *cognitoidentity.Client {
		return cognitoidentity.NewFromConfig(*c, func(o *cognitoidentity.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Cognito Identity clients: %w", err)
	}

	return &CognitoIdentityPoolCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*CognitoIdentityPoolCollector) Name() string {
	return "cognito_identity_pool"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*CognitoIdentityPoolCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*CognitoIdentityPoolCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "SubCategory2", Value: func(r Resource) string { return r.SubCategory2 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID
		{Header: "AllowUnauthenticated", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AllowUnauthenticated") }},
		{Header: "DeveloperProviderName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DeveloperProviderName") }},
		{Header: "SupportedLoginProviders", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SupportedLoginProviders") }},
		{Header: "CognitoIdentityProviders", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CognitoIdentityProviders") }},
		{Header: "OpenIdConnectProviderARNs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "OpenIdConnectProviderARNs") }},
		{Header: "SamlProviderARNs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SamlProviderARNs") }},
	}
}

// Collect collects Cognito Identity Pool resources for the specified region.
// The collector must have been initialized with clients for this region.
func (c *CognitoIdentityPoolCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	client, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	resources, err := collectIdentityPools(ctx, region, client)
	if err != nil {
		return nil, fmt.Errorf("failed to collect identity pools: %w", err)
	}
	return resources, nil
}

// collectIdentityPools lists identity pools and returns resources.
func collectIdentityPools(ctx context.Context, region string, identitySvc *cognitoidentity.Client) ([]Resource, error) {
	var resources []Resource
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
			var developerProviderName string
			var supportedLoginProviders []string
			var cognitoIdentityProviders []string
			var openIdConnectProviderARNs []string
			var samlProviderARNs []string
			desc, descErr := identitySvc.DescribeIdentityPool(ctx, &cognitoidentity.DescribeIdentityPoolInput{
				IdentityPoolId: pool.IdentityPoolId,
			})
			if descErr == nil && desc != nil {
				allowUnauth = desc.AllowUnauthenticatedIdentities
				developerProviderName = helpers.StringValue(desc.DeveloperProviderName)
				// SupportedLoginProviders is a map[string]string; convert to key=value slices
				for k, v := range desc.SupportedLoginProviders {
					supportedLoginProviders = append(supportedLoginProviders, fmt.Sprintf("%s=%s", k, v))
				}
				for j := range desc.CognitoIdentityProviders {
					cip := desc.CognitoIdentityProviders[j]
					cognitoIdentityProviders = append(cognitoIdentityProviders, fmt.Sprintf("%s=%s", helpers.StringValue(cip.ProviderName), helpers.StringValue(cip.ClientId)))
				}
				// slices are already the desired type; assign directly
				openIdConnectProviderARNs = desc.OpenIdConnectProviderARNs
				samlProviderARNs = desc.SamlProviderARNs
			}
			resources = append(resources, NewResource(&ResourceInput{
				Category:     "cognito",
				SubCategory1: "IdentityPool",
				Name:         pool.IdentityPoolName,
				Region:       region,
				ARN:          pool.IdentityPoolId,
				RawData: map[string]any{
					"AllowUnauthenticated":      allowUnauth,
					"DeveloperProviderName":     developerProviderName,
					"SupportedLoginProviders":   supportedLoginProviders,
					"CognitoIdentityProviders":  cognitoIdentityProviders,
					"OpenIdConnectProviderARNs": openIdConnectProviderARNs,
					"SamlProviderARNs":          samlProviderARNs,
				},
			}))
		}

		if out.NextToken == nil {
			break
		}
		nextToken = out.NextToken
	}
	return resources, nil
}
