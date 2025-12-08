package resources

import (
	"context"
	"fmt"

	"github.com/y-miyazaki/arc/internal/aws/helpers"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/quicksight"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// QuickSightCollector collects QuickSight Data Sources and Analyses.
// It retrieves details such as type, status, and creation date.
// The collector requires AWS Account ID to list resources, which is retrieved via STS.
// It handles both Data Sources and Analyses, providing a comprehensive view of QuickSight assets.
type QuickSightCollector struct{}

// Name returns the collector name.
func (*QuickSightCollector) Name() string {
	return "quicksight"
}

// ShouldSort returns true.
func (*QuickSightCollector) ShouldSort() bool {
	return true
}

// GetColumns returns the CSV column definitions for QuickSight.
func (*QuickSightCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "CreatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedDate") }},
	}
}

// Collect collects QuickSight resources from the specified region.
func (*QuickSightCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	qsSvc := quicksight.NewFromConfig(*cfg, func(o *quicksight.Options) {
		o.Region = region
	})
	stsSvc := sts.NewFromConfig(*cfg)

	// Get Account ID
	identity, err := stsSvc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity: %w", err)
	}
	accountID := aws.ToString(identity.Account)

	var resources []Resource

	// Data Sources
	dsPaginator := quicksight.NewListDataSourcesPaginator(qsSvc, &quicksight.ListDataSourcesInput{
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
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "quicksight",
				SubCategory: "DataSource",
				Name:        ds.Name,
				Region:      region,
				ARN:         ds.DataSourceId,
				RawData: map[string]any{
					"Type":   ds.Type,
					"Status": ds.Status,
				},
			}))
		}
	}

	// Analyses
	anPaginator := quicksight.NewListAnalysesPaginator(qsSvc, &quicksight.ListAnalysesInput{
		AwsAccountId: &accountID,
	})
	for anPaginator.HasMorePages() {
		page, anErr := anPaginator.NextPage(ctx)
		if anErr != nil {
			break
		}

		for i := range page.AnalysisSummaryList {
			an := &page.AnalysisSummaryList[i]
			resources = append(resources, NewResource(&ResourceInput{
				Category:    "quicksight",
				SubCategory: "Analysis",
				Name:        an.Name,
				Region:      region,
				ARN:         an.AnalysisId,
				RawData: map[string]any{
					"Status":      an.Status,
					"CreatedDate": an.CreatedTime,
				},
			}))
		}
	}

	return resources, nil //nolint:nilerr
}
