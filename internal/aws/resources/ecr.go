// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// ECRCollector collects ECR resources.
// It uses dependency injection to manage ECR clients for multiple regions.
type ECRCollector struct {
	clients      map[string]*ecr.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewECRCollector creates a new ECR collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create ECR clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *ECRCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewECRCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*ECRCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *ecr.Client {
		return ecr.NewFromConfig(*c, func(o *ecr.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ECR clients: %w", err)
	}

	return &ECRCollector{
		clients:      clients,
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*ECRCollector) Name() string {
	return "ecr"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*ECRCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*ECRCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "URI", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "URI") }},
		{Header: "Mutability", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Mutability") }},
		{Header: "Encryption", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Encryption") }},
		{Header: "KMSKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KMSKey") }},
		{Header: "ScanOnPush", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ScanOnPush") }},
		{Header: "LifecyclePolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LifecyclePolicy") }},
		{Header: "ImageCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ImageCount") }},
		{Header: "CreatedAt", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedAt") }},
	}
}

// Collect collects ECR resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *ECRCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	var resources []Resource

	paginator := ecr.NewDescribeRepositoriesPaginator(svc, &ecr.DescribeRepositoriesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe repositories: %w", err)
		}

		for i := range page.Repositories {
			repo := &page.Repositories[i]
			// Get image count
			// Note: describe-images can be expensive if there are many images.
			// The bash script does `aws ecr describe-images ... | length`.
			// We can use ListImages or DescribeImages with max results 1 to just check existence?
			// No, we need the count.
			// Ideally we should use ListImages which is lighter than DescribeImages if we just want count?
			// But ListImages is paginated.
			// DescribeImages is also paginated.
			// The bash script uses `describe-images --output json` then `jq .imageDetails | length`.
			// This implies it counts all images.
			// For now, let's try to list images and count them.
			// To avoid too many API calls, maybe we can skip or optimize?
			// But to be faithful to the script, we should count.
			// Let's use ListImagesPaginator to count.

			count := 0
			imgPaginator := ecr.NewListImagesPaginator(svc, &ecr.ListImagesInput{
				RepositoryName: repo.RepositoryName,
			})
			for imgPaginator.HasMorePages() {
				imgPage, imgErr := imgPaginator.NextPage(ctx)
				if imgErr != nil {
					// If we fail to list images, just assume 0 or log error?
					// Bash script defaults to '0' on error.
					count = 0
					break
				}
				count += len(imgPage.ImageIds)
			}

			encryption := "NONE"
			kmsKey := ""
			scanOnPush := "false"
			if repo.EncryptionConfiguration != nil {
				encryption = string(repo.EncryptionConfiguration.EncryptionType)
				if repo.EncryptionConfiguration.KmsKey != nil {
					kmsKey = *repo.EncryptionConfiguration.KmsKey
				}
			}
			if repo.ImageScanningConfiguration != nil {
				scanOnPush = strconv.FormatBool(repo.ImageScanningConfiguration.ScanOnPush)
			}

			// Get lifecycle policy
			var lifecyclePolicy string
			if lifecycleOut, lifecycleErr := svc.GetLifecyclePolicy(ctx, &ecr.GetLifecyclePolicyInput{
				RepositoryName: repo.RepositoryName,
			}); lifecycleErr == nil {
				if lifecycleOut.LifecyclePolicyText != nil {
					// Parse and format JSON for better readability
					rawJSON := *lifecycleOut.LifecyclePolicyText
					if formatted, formatErr := helpers.FormatJSONIndent(rawJSON); formatErr == nil {
						lifecyclePolicy = formatted
					} else {
						// Fallback to raw JSON if formatting fails
						lifecyclePolicy = rawJSON
					}
				}
			}
			resources = append(resources, NewResource(&ResourceInput{
				Category:     "ecr",
				SubCategory1: "Repository",
				Name:         repo.RepositoryName,
				Region:       region,
				ARN:          repo.RepositoryArn,
				RawData: map[string]any{
					"URI":             repo.RepositoryUri,
					"Mutability":      repo.ImageTagMutability,
					"Encryption":      encryption,
					"KMSKey":          kmsKey,
					"ScanOnPush":      scanOnPush,
					"LifecyclePolicy": lifecyclePolicy,
					"ImageCount":      strconv.Itoa(count),
					"CreatedAt":       repo.CreatedAt,
				},
			}))
		}
	}

	return resources, nil
}
