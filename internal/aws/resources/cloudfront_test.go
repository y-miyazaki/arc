package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCloudFrontCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewCloudFrontCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewCloudFrontCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewCloudFrontCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestCloudFrontCollector_Basic(t *testing.T) {
	collector := &CloudFrontCollector{
		clients: make(map[string]*cloudfront.Client),
	}
	assert.Equal(t, "cloudfront", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCloudFrontCollector_Collect_NoClient(t *testing.T) {
	collector := &CloudFrontCollector{
		clients: make(map[string]*cloudfront.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-east-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestCloudFrontCollector_GetColumns(t *testing.T) {
	collector := &CloudFrontCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ID",
		"AlternateDomain", "Origin", "PriceClass", "WAF", "Status",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "CloudFront",
		SubCategory:    "Distribution",
		SubSubCategory: "",
		Name:           "test-distribution",
		Region:         "us-east-1",
		ARN:            "",
		RawData: map[string]interface{}{
			"ID":              "E1A2B3C4D5F6G",
			"AlternateDomain": "cdn.example.com",
			"Origin":          "example.s3.amazonaws.com",
			"PriceClass":      "PriceClass_100",
			"WAF":             "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/test-waf/12345678-1234-1234-1234-123456789012",
			"Status":          "Deployed",
		},
	}

	expectedValues := []string{
		"CloudFront", "Distribution", "", "test-distribution", "us-east-1", "E1A2B3C4D5F6G",
		"cdn.example.com", "example.s3.amazonaws.com", "PriceClass_100", "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/test-waf/12345678-1234-1234-1234-123456789012", "Deployed",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
