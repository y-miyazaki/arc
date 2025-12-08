package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockCloudFrontCollector is a testable version of CloudFrontCollector that uses mock data
type MockCloudFrontCollector struct{}

func NewMockCloudFrontCollector() *MockCloudFrontCollector {
	return &MockCloudFrontCollector{}
}

func (c *MockCloudFrontCollector) Name() string {
	return "cloudfront"
}

func (c *MockCloudFrontCollector) ShouldSort() bool {
	return true
}

func (c *MockCloudFrontCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "AlternateDomain", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AlternateDomain") }},
		{Header: "Origin", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Origin") }},
		{Header: "PriceClass", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PriceClass") }},
		{Header: "WAF", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "WAF") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
	}
}

func (c *MockCloudFrontCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// CloudFront is a global service, only process from us-east-1
	if region != "us-east-1" {
		return nil, nil
	}

	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Distribution
	r1 := Resource{
		Category:    "cloudfront",
		SubCategory: "Distribution",
		Name:        "d1234567890abcdef0.cloudfront.net",
		Region:      "Global",
		RawData: helpers.NormalizeRawData(map[string]any{
			"ID":              "E1A2B3C4D5F6G7H8",
			"AlternateDomain": []string{"example.com", "www.example.com"},
			"Origin":          "my-bucket.s3.amazonaws.com",
			"PriceClass":      "PriceClass_100",
			"WAF":             "MyWebACL",
			"Status":          "Deployed",
		}),
	}
	resources = append(resources, r1)

	return resources, nil
}

func TestCloudFrontCollector_Basic(t *testing.T) {
	collector := &CloudFrontCollector{}
	assert.Equal(t, "cloudfront", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCloudFrontCollector_GetColumns(t *testing.T) {
	collector := &CloudFrontCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "AlternateDomain", "Origin", "PriceClass", "WAF", "Status",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "Content Delivery",
		SubCategory:    "CloudFront",
		SubSubCategory: "Distribution",
		Name:           "d1234567890abcdef0.cloudfront.net",
		Region:         "Global",
		RawData: map[string]any{
			"ID":              "E1A2B3C4D5F6G7H8",
			"AlternateDomain": "example.com, www.example.com",
			"Origin":          "my-bucket.s3.amazonaws.com",
			"PriceClass":      "PriceClass_100",
			"WAF":             "MyWebACL",
			"Status":          "Deployed",
		},
	}

	expectedValues := []string{
		"Content Delivery", "CloudFront", "Distribution", "d1234567890abcdef0.cloudfront.net", "Global",
		"E1A2B3C4D5F6G7H8", "example.com, www.example.com", "my-bucket.s3.amazonaws.com", "PriceClass_100", "MyWebACL", "Deployed",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockCloudFrontCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockCloudFrontCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 1)

	// Check resource (Distribution)
	r1 := resources[0]
	assert.Equal(t, "cloudfront", r1.Category)
	assert.Equal(t, "Distribution", r1.SubCategory)
	assert.Equal(t, "d1234567890abcdef0.cloudfront.net", r1.Name)
	assert.Equal(t, "Global", r1.Region)
	assert.Equal(t, "E1A2B3C4D5F6G7H8", helpers.GetMapValue(r1.RawData, "ID"))
	assert.Equal(t, "example.com\nwww.example.com", helpers.GetMapValue(r1.RawData, "AlternateDomain"))
	assert.Equal(t, "my-bucket.s3.amazonaws.com", helpers.GetMapValue(r1.RawData, "Origin"))
	assert.Equal(t, "PriceClass_100", helpers.GetMapValue(r1.RawData, "PriceClass"))
	assert.Equal(t, "MyWebACL", helpers.GetMapValue(r1.RawData, "WAF"))
	assert.Equal(t, "Deployed", helpers.GetMapValue(r1.RawData, "Status"))
}

func TestMockCloudFrontCollector_Collect_NonUSEast1(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "eu-west-1"

	collector := NewMockCloudFrontCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 0)
}
