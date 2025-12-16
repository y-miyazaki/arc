// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// SecretsManagerCollector collects Secrets Manager secrets.
// It uses dependency injection to manage Secrets Manager clients for multiple regions.
type SecretsManagerCollector struct {
	clients      map[string]*secretsmanager.Client
	nameResolver *helpers.NameResolver
}

// NewSecretsManagerCollector creates a new Secrets Manager collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create Secrets Manager clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *SecretsManagerCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewSecretsManagerCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*SecretsManagerCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *secretsmanager.Client {
		return secretsmanager.NewFromConfig(*c, func(o *secretsmanager.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Secrets Manager clients: %w", err)
	}

	return &SecretsManagerCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*SecretsManagerCollector) Name() string {
	return "secretsmanager"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*SecretsManagerCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*SecretsManagerCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "RotationEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RotationEnabled") }},
		{Header: "RotationLambdaARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RotationLambdaARN") }},
		{Header: "SecretString", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecretString") }},
		{Header: "LastAccessedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastAccessedDate") }},
		{Header: "LastRotatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastRotatedDate") }},
		{Header: "LastChangedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastChangedDate") }},
	}
}

// Collect collects Secrets Manager resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *SecretsManagerCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// Get all KMS keys to resolve names efficiently
	kmsMap, err := c.nameResolver.GetAllKMSKeys(ctx, region)
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

			// Get secret value to retrieve SecretString
			var secretStringValue string
			if secret.ARN != nil {
				getValueOutput, getValueErr := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
					SecretId: secret.ARN,
				})
				if getValueErr == nil && getValueOutput.SecretString != nil { // pragma: allowlist secret
					// Format as indented JSON if valid, otherwise return raw string
					secretStringValue = helpers.FormatJSONIndentOrRaw(*getValueOutput.SecretString)
				}
			}

			r := NewResource(&ResourceInput{
				Category:     "secretsmanager",
				SubCategory1: "Secret",
				Name:         secret.Name,
				Region:       region,
				ARN:          secret.ARN,
				RawData: map[string]any{
					"Description":       secret.Description,
					"KmsKey":            helpers.ResolveNameFromMap(secret.KmsKeyId, kmsMap),
					"RotationEnabled":   secret.RotationEnabled,
					"RotationLambdaARN": secret.RotationLambdaARN,
					"SecretString":      secretStringValue,
					"LastAccessedDate":  secret.LastAccessedDate,
					"LastRotatedDate":   secret.LastRotatedDate,
					"LastChangedDate":   secret.LastChangedDate,
				},
			})
			resources = append(resources, r)
		}
	}

	return resources, nil
}
