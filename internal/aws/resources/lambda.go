// Package resources provides AWS resource collectors for different services.
package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// LambdaCollector collects Lambda functions
type LambdaCollector struct{}

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

// Collect collects Lambda resources.
func (*LambdaCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := lambda.NewFromConfig(*cfg, func(o *lambda.Options) {
		o.Region = region
	})

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
			var envVarsStr string
			if function.Environment != nil && function.Environment.Variables != nil {
				var envEntries []string
				for k, v := range function.Environment.Variables {
					// Mask private keys
					if strings.Contains(k, "PRIVATE_KEY") {
						v = "*****"
					}
					envEntries = append(envEntries, fmt.Sprintf("%s=%s", k, v))
				}
				// Sort for deterministic output
				sort.Strings(envEntries)
				envVarsStr = strings.Join(envEntries, "\n")
			}

			resources = append(resources, NewResource(&ResourceInput{
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
					"EnvVars":      envVarsStr,
					"LastModified": function.LastModified,
				},
			}))
		}
	}

	return resources, nil
}
