// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// KMSCollector collects KMS keys.
// It uses dependency injection to manage KMS clients for multiple regions.
type KMSCollector struct {
	clients      map[string]*kms.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewKMSCollector creates a new KMS collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create KMS clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *KMSCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewKMSCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*KMSCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *kms.Client {
		return kms.NewFromConfig(*c, func(o *kms.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS clients: %w", err)
	}

	return &KMSCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the collector name.
func (*KMSCollector) Name() string {
	return "kms"
}

// ShouldSort returns false to preserve key order.
func (*KMSCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for KMS keys.
func (*KMSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "KeyUsage", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KeyUsage") }},
		{Header: "KeyManager", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KeyManager") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

// Collect collects KMS keys for the specified region.
// The collector must have been initialized with a client for this region.
func (c *KMSCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	// List all KMS keys
	listPaginator := kms.NewListKeysPaginator(svc, &kms.ListKeysInput{})
	for listPaginator.HasMorePages() {
		listPage, err := listPaginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list KMS keys: %w", err)
		}

		for i := range listPage.Keys {
			key := &listPage.Keys[i]

			// Get detailed key information
			var describeOut *kms.DescribeKeyOutput
			describeOut, err = svc.DescribeKey(ctx, &kms.DescribeKeyInput{
				KeyId: key.KeyId,
			})
			if err != nil {
				// Skip keys that cannot be described (e.g., deleted keys)
				continue
			}

			keyMetadata := describeOut.KeyMetadata

			// Get key aliases
			var keyName string
			var aliasesOut *kms.ListAliasesOutput
			aliasesOut, err = svc.ListAliases(ctx, &kms.ListAliasesInput{
				KeyId: key.KeyId,
			})
			if err == nil && aliasesOut != nil && len(aliasesOut.Aliases) > 0 {
				keyName = *aliasesOut.Aliases[0].AliasName
			} else {
				keyName = *key.KeyId
			}

			resources = append(resources, NewResource(&ResourceInput{
				Category:     "kms",
				SubCategory1: "Key",
				Name:         keyName,
				Region:       region,
				ARN:          keyMetadata.Arn,
				RawData: map[string]any{
					"Description": keyMetadata.Description,
					"KeyUsage":    keyMetadata.KeyUsage,
					"KeyManager":  keyMetadata.KeyManager,
					"State":       keyMetadata.KeyState,
				},
			}))
		}
	}

	return resources, nil
}
