// Package resources contains AWS resource collectors including API Gateway.
// This package provides collectors for various AWS services, each implementing
// the Collector interface with dependency injection for regional clients.
// API Gateway collector specifically handles REST APIs (v1) and HTTP APIs (v2).
package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// Sentinel errors for API Gateway operations.
var (
	ErrNoAPIGatewayV1Client = errors.New("no API Gateway v1 client found for region")
	ErrNoAPIGatewayV2Client = errors.New("no API Gateway v2 client found for region")
)

// APIGatewayCollector collects API Gateway resources (REST and HTTP APIs).
type APIGatewayCollector struct {
	clientsV1    map[string]*apigateway.Client
	clientsV2    map[string]*apigatewayv2.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewAPIGatewayCollector creates a new API Gateway collector with regional clients.
func NewAPIGatewayCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*APIGatewayCollector, error) {
	clientsV1, err := helpers.CreateRegionalClients(cfg, regions,
		func(cfg *aws.Config, region string) *apigateway.Client {
			return apigateway.NewFromConfig(*cfg, func(o *apigateway.Options) {
				o.Region = region
			})
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create API Gateway v1 clients: %w", err)
	}

	clientsV2, err := helpers.CreateRegionalClients(cfg, regions,
		func(cfg *aws.Config, region string) *apigatewayv2.Client {
			return apigatewayv2.NewFromConfig(*cfg, func(o *apigatewayv2.Options) {
				o.Region = region
			})
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create API Gateway v2 clients: %w", err)
	}

	return &APIGatewayCollector{
		clientsV1:    clientsV1,
		clientsV2:    clientsV2,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the collector name.
func (*APIGatewayCollector) Name() string {
	return "apigateway"
}

// ShouldSort returns false to maintain API and authorizer grouping.
func (*APIGatewayCollector) ShouldSort() bool {
	return false
}

// GetColumns returns the CSV column definitions for API Gateway.
func (*APIGatewayCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "ProtocolType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ProtocolType") }},
		{Header: "WAF", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "WAF") }},
		{Header: "AuthorizerType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AuthorizerType") }},
		{Header: "AuthorizerProviderARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AuthorizerProviderARN") }},
		{Header: "CreatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedDate") }},
	}
}

// Collect collects API Gateway resources from the specified region.
func (c *APIGatewayCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svcV1, okV1 := c.clientsV1[region]
	if !okV1 {
		return nil, fmt.Errorf("%w: %s", ErrNoAPIGatewayV1Client, region)
	}

	svcV2, okV2 := c.clientsV2[region]
	if !okV2 {
		return nil, fmt.Errorf("%w: %s", ErrNoAPIGatewayV2Client, region)
	}

	var resources []Resource

	// 1. REST APIs (v1)
	paginatorV1 := apigateway.NewGetRestApisPaginator(svcV1, &apigateway.GetRestApisInput{})
	for paginatorV1.HasMorePages() {
		page, err := paginatorV1.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get REST APIs: %w", err)
		}

		for i := range page.Items {
			api := &page.Items[i]

			// Get Stages to find WAF
			var wafName string
			stages, stageErr := svcV1.GetStages(ctx, &apigateway.GetStagesInput{
				RestApiId: api.Id,
			})
			if stageErr == nil {
				for j := range stages.Item {
					stage := &stages.Item[j]
					if stage.WebAclArn != nil {
						// Extract WAF name from ARN
						arn := *stage.WebAclArn
						if idx := strings.Index(arn, "/webacl/"); idx != -1 {
							sub := arn[idx+len("/webacl/"):]
							parts := strings.SplitN(sub, "/", 2) //nolint:mnd
							if len(parts) > 0 {
								wafName = parts[0]
							}
						} else {
							// Fallback
							parts := strings.Split(arn, "/")
							if len(parts) > 0 {
								wafName = parts[len(parts)-1]
							}
						}
						break
					}
				}
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:    "apigateway",
				SubCategory: "RestAPI",
				Name:        api.Name,
				Region:      region,
				ARN:         "",
				RawData: map[string]any{
					"Description":  api.Description,
					"ID":           api.Id,
					"ProtocolType": "REST",
					"WAF":          wafName,
					"CreatedDate":  api.CreatedDate,
				},
			}))

			// Get Authorizers
			auths, authErr := svcV1.GetAuthorizers(ctx, &apigateway.GetAuthorizersInput{
				RestApiId: api.Id,
			})
			if authErr == nil {
				for j := range auths.Items {
					auth := &auths.Items[j]
					var providerARNs []string
					providerARNs = append(providerARNs, auth.ProviderARNs...)
					if auth.AuthorizerUri != nil {
						providerARNs = append(providerARNs, *auth.AuthorizerUri)
					}

					resources = append(resources, NewResource(&ResourceInput{
						Category:       "apigateway",
						SubCategory:    "",
						SubSubCategory: "Authorizer",
						Name:           auth.Name,
						Region:         region,
						ARN:            "",
						RawData: map[string]any{
							"AuthorizerType":        auth.Type,
							"AuthorizerProviderARN": providerARNs,
						},
					}))
				}
			}
		}
	}

	// 2. HTTP APIs (v2)
	var nextToken *string
	for {
		apisV2, err := svcV2.GetApis(ctx, &apigatewayv2.GetApisInput{
			NextToken: nextToken,
		})
		if err != nil {
			break
		}

		for i := range apisV2.Items {
			api := &apisV2.Items[i]

			// WAF extraction for HTTP APIs is skipped as it's not straightforward in SDK v2 struct
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "apigateway",
				SubCategory: "HttpAPI",
				Name:        api.Name,
				Region:      region,
				ARN:         "",
				RawData: map[string]any{
					"Description":  api.Description,
					"ID":           api.ApiId,
					"ProtocolType": api.ProtocolType,
					"CreatedDate":  api.CreatedDate,
				},
			}))

			// Get Authorizers
			var authToken *string
			for {
				authsV2, authErr := svcV2.GetAuthorizers(ctx, &apigatewayv2.GetAuthorizersInput{
					ApiId:     api.ApiId,
					NextToken: authToken,
				})
				if authErr != nil {
					break
				}
				for j := range authsV2.Items {
					auth := &authsV2.Items[j]
					var providerInfo []string
					providerInfo = append(providerInfo, auth.IdentitySource...)
					if auth.AuthorizerUri != nil {
						providerInfo = append(providerInfo, *auth.AuthorizerUri)
					}
					if auth.AuthorizerCredentialsArn != nil {
						providerInfo = append(providerInfo, *auth.AuthorizerCredentialsArn)
					}
					resources = append(resources, NewResource(&ResourceInput{
						Category:       "apigateway",
						SubCategory:    "",
						SubSubCategory: "Authorizer",
						Name:           auth.Name,
						Region:         region,
						ARN:            "",
						RawData: map[string]any{
							"AuthorizerType":        auth.AuthorizerType,
							"AuthorizerProviderARN": providerInfo,
						},
					}))
				}
				if authsV2.NextToken == nil {
					break
				}
				authToken = authsV2.NextToken
			}
		}

		if apisV2.NextToken == nil {
			break
		}
		nextToken = apisV2.NextToken
	}

	return resources, nil //nolint:nilerr
}
