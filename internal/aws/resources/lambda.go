// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// LambdaCollector collects Lambda functions.
// It uses dependency injection to manage Lambda clients for multiple regions.
type LambdaCollector struct {
	clients      map[string]*lambda.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewLambdaCollector creates a new Lambda collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Lambda clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *LambdaCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewLambdaCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*LambdaCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *lambda.Client {
		return lambda.NewFromConfig(*c, func(o *lambda.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Lambda clients: %w", err)
	}

	return &LambdaCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*LambdaCollector) Name() string {
	return "lambda"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*LambdaCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*LambdaCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Runtime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Runtime") }},
		{Header: "Architecture", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Architecture") }},
		{Header: "MemorySize", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MemorySize") }},
		{Header: "Timeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Timeout") }},
		{Header: "EnvVars", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EnvVars") }},
		{Header: "LastModified", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastModified") }},
	}
}

// Collect collects Lambda resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *LambdaCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// List functions with pagination
	paginator := lambda.NewListFunctionsPaginator(svc, &lambda.ListFunctionsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list functions: %w", err)
		}

		for i := range page.Functions {
			function := &page.Functions[i]

			architecture := ""
			if len(function.Architectures) > 0 {
				architecture = string(function.Architectures[0])
			}

			// Process environment variables
			var envEntries []string
			if function.Environment != nil && function.Environment.Variables != nil {
				for k, v := range function.Environment.Variables {
					// Mask private keys
					if strings.Contains(k, "PRIVATE_KEY") {
						v = "*****"
					}
					envEntries = append(envEntries, fmt.Sprintf("%s=%s", k, v))
				}
			}

			r := NewResource(&ResourceInput{
				Category:    "lambda",
				SubCategory: "Function",
				Name:        function.FunctionName,
				Region:      region,
				ARN:         function.FunctionArn,
				RawData: map[string]any{
					"RoleARN":      function.Role,
					"Type":         "Function",
					"Runtime":      function.Runtime,
					"Architecture": architecture,
					"MemorySize":   function.MemorySize,
					"Timeout":      function.Timeout,
					"EnvVars":      envEntries,
					"LastModified": function.LastModified,
				},
			})
			resources = append(resources, r)
		}
	}

	return resources, nil
}
