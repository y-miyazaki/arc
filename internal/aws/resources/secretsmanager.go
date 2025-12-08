// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SecretsManagerCollector collects Secrets Manager secrets.
type SecretsManagerCollector struct{}

// Name returns the collector name.
func (*SecretsManagerCollector) Name() string {
	return "secretsmanager"
}

// ShouldSort returns true.
func (*SecretsManagerCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for Secrets Manager secrets.
func (*SecretsManagerCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "RotationEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RotationEnabled") }},
		{Header: "RotationLambdaARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RotationLambdaARN") }},
		{Header: "LastAccessedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastAccessedDate") }},
		{Header: "LastRotatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastRotatedDate") }},
		{Header: "LastChangedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastChangedDate") }},
	}
}

// Collect collects Secrets Manager secrets from the specified region.
func (*SecretsManagerCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	svc := secretsmanager.NewFromConfig(*cfg, func(o *secretsmanager.Options) {
		o.Region = region
	})

	var resources []Resource

	// Get all KMS keys to resolve names efficiently
	kmsMap, err := helpers.GetAllKMSKeys(ctx, cfg, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}

	paginator := secretsmanager.NewListSecretsPaginator(svc, &secretsmanager.ListSecretsInput{})
	for paginator.HasMorePages() {
		page, pageErr := paginator.NextPage(ctx)
		if pageErr != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", pageErr)
		}

		for i := range page.SecretList {
			secret := &page.SecretList[i]
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "secretsmanager",
				SubCategory: "Secret",
				Name:        secret.Name,
				Region:      region,
				ARN:         secret.ARN,
				RawData: map[string]any{
					"Description":       secret.Description,
					"KmsKey":            helpers.ResolveNameFromMap(secret.KmsKeyId, kmsMap),
					"RotationEnabled":   secret.RotationEnabled,
					"RotationLambdaARN": secret.RotationLambdaARN,
					"LastAccessedDate":  secret.LastAccessedDate,
					"LastRotatedDate":   secret.LastRotatedDate,
					"LastChangedDate":   secret.LastChangedDate,
				},
			}))
		}
	}

	return resources, nil
}
