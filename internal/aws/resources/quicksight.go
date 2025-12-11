// Package resources provides AWS resource collectors.
package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/quicksight"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// QuickSightCollector collects QuickSight Data Sources and Analyses.
// It uses dependency injection to manage QuickSight clients for multiple regions.
type QuickSightCollector struct {
	clients      map[string]*quicksight.Client
	stsClient    *sts.Client
	nameResolver *helpers.NameResolver //nolint:unused // Reserved for future resource name resolution
}

// NewQuickSightCollector creates a new QuickSight collector with clients for the specified regions.
// This constructor follows the standard naming convention for dependency injection:
// New<ServiceName>Collector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*<ServiceName>Collector, error)
//
// Parameters:
//   - cfg: AWS configuration with credentials
//   - regions: List of AWS regions to create QuickSight clients for
//   - nameResolver: Shared NameResolver instance for resource name resolution
//
// Returns:
//   - *QuickSightCollector: Initialized collector with regional clients and name resolver
//   - error: Error if client creation fails
func NewQuickSightCollector(cfg *aws.Config, regions []string, nameResolver *helpers.NameResolver) (*QuickSightCollector, error) {
	clients, err := helpers.CreateRegionalClients(cfg, regions, func(c *aws.Config, region string) *quicksight.Client {
		return quicksight.NewFromConfig(*c, func(o *quicksight.Options) {
			o.Region = region
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create QuickSight clients: %w", err)
	}

	return &QuickSightCollector{
		clients:      clients,
		stsClient:    sts.NewFromConfig(*cfg),
		nameResolver: nameResolver,
	}, nil
}

// Name returns the resource name of the collector.
func (*QuickSightCollector) Name() string {
	return "quicksight"
}

// ShouldSort returns whether the collected resources should be sorted.
func (*QuickSightCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV columns for the collector.
func (*QuickSightCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory1", Value: func(r Resource) string { return r.SubCategory1 }},
		{Header: "SubCategory2", Value: func(r Resource) string { return r.SubCategory2 }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "CreatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedDate") }},
	}
}

// Collect collects QuickSight resources for the specified region.
// The collector must have been initialized with a client for this region.
func (c *QuickSightCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	svc, ok := c.clients[region]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoClientForRegion, region)
	}

	// Get Account ID
	identity, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity: %w", err)
	}
	accountID := aws.ToString(identity.Account)

	var resources []Resource

	// Data Sources
	dsPaginator := quicksight.NewListDataSourcesPaginator(svc, &quicksight.ListDataSourcesInput{
		AwsAccountId: &accountID,
	})
	for dsPaginator.HasMorePages() {
		page, dsErr := dsPaginator.NextPage(ctx)
		if dsErr != nil {
			// QuickSight might not be subscribed or available
			break
		}

		for i := range page.DataSources {
			ds := &page.DataSources[i]
			r := NewResource(&ResourceInput{
				Category:     "quicksight",
				SubCategory1: "DataSource",
				Name:         ds.Name,
				Region:       region,
				ARN:          ds.DataSourceId,
				RawData: map[string]any{
					"Type":   ds.Type,
					"Status": ds.Status,
				},
			})
			resources = append(resources, r)
		}
	}

	// Analyses
	anPaginator := quicksight.NewListAnalysesPaginator(svc, &quicksight.ListAnalysesInput{
		AwsAccountId: &accountID,
	})
	for anPaginator.HasMorePages() {
		page, anErr := anPaginator.NextPage(ctx)
		if anErr != nil {
			break
		}

		for i := range page.AnalysisSummaryList {
			an := &page.AnalysisSummaryList[i]
			r := NewResource(&ResourceInput{
				Category:     "quicksight",
				SubCategory1: "Analysis",
				Name:         an.Name,
				Region:       region,
				ARN:          an.AnalysisId,
				RawData: map[string]any{
					"Status":      an.Status,
					"CreatedDate": an.CreatedTime,
				},
			})
			resources = append(resources, r)
		}
	}

	return resources, nil //nolint:nilerr
}
