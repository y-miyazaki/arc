package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockQuickSightCollector is a testable version of QuickSightCollector that uses mock data
type MockQuickSightCollector struct{}

func NewMockQuickSightCollector() *MockQuickSightCollector {
	return &MockQuickSightCollector{}
}

func (c *MockQuickSightCollector) Name() string {
	return "quicksight"
}

func (c *MockQuickSightCollector) ShouldSort() bool {
	return true
}

func (c *MockQuickSightCollector) GetColumns() []Column {
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

func (c *MockQuickSightCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Data Source
	r1 := Resource{
		Category:    "quicksight",
		SubCategory: "DataSource",
		Name:        "My Redshift Data Source",
		Region:      region,
		ARN:         "datasource-12345",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Type":   "REDSHIFT",
			"Status": "CREATION_SUCCESSFUL",
		}),
	}
	resources = append(resources, r1)

	// Mock Analysis
	r2 := Resource{
		Category:    "quicksight",
		SubCategory: "Analysis",
		Name:        "Sales Dashboard Analysis",
		Region:      region,
		ARN:         "analysis-67890",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Status":      "CREATION_SUCCESSFUL",
			"CreatedDate": time.Date(2023, 8, 15, 10, 30, 0, 0, time.UTC),
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestQuickSightCollector_Basic(t *testing.T) {
	collector := &QuickSightCollector{}
	assert.Equal(t, "quicksight", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestQuickSightCollector_GetColumns(t *testing.T) {
	collector := &QuickSightCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "Type", "Status", "CreatedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "quicksight",
		SubCategory:    "Analysis",
		SubSubCategory: "",
		Name:           "Sales Analysis",
		Region:         "us-east-1",
		ARN:            "analysis-12345",
		RawData: map[string]any{
			"Type":        "ANALYSIS",
			"Status":      "CREATION_SUCCESSFUL",
			"CreatedDate": "2023-08-15T10:30:00Z",
		},
	}

	expectedValues := []string{
		"quicksight", "Analysis", "", "Sales Analysis", "us-east-1",
		"analysis-12345", "ANALYSIS", "CREATION_SUCCESSFUL", "2023-08-15T10:30:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockQuickSightCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockQuickSightCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (Data Source)
	r1 := resources[0]
	assert.Equal(t, "quicksight", r1.Category)
	assert.Equal(t, "DataSource", r1.SubCategory)
	assert.Equal(t, "My Redshift Data Source", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "datasource-12345", r1.ARN)
	assert.Equal(t, "REDSHIFT", helpers.GetMapValue(r1.RawData, "Type"))
	assert.Equal(t, "CREATION_SUCCESSFUL", helpers.GetMapValue(r1.RawData, "Status"))

	// Check second resource (Analysis)
	r2 := resources[1]
	assert.Equal(t, "quicksight", r2.Category)
	assert.Equal(t, "Analysis", r2.SubCategory)
	assert.Equal(t, "Sales Dashboard Analysis", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "analysis-67890", r2.ARN)
	assert.Equal(t, "CREATION_SUCCESSFUL", helpers.GetMapValue(r2.RawData, "Status"))
	assert.Equal(t, "2023-08-15T10:30:00Z", helpers.GetMapValue(r2.RawData, "CreatedDate"))
}
