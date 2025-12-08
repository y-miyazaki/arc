package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewAPIGatewayCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewAPIGatewayCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clientsV1, 2)
	assert.Len(t, collector.clientsV2, 2)
	assert.Contains(t, collector.clientsV1, "us-east-1")
	assert.Contains(t, collector.clientsV1, "eu-west-1")
	assert.Contains(t, collector.clientsV2, "us-east-1")
	assert.Contains(t, collector.clientsV2, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewAPIGatewayCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewAPIGatewayCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clientsV1)
	assert.Empty(t, collector.clientsV2)
	assert.NotNil(t, collector.nameResolver)
}

func TestAPIGatewayCollector_Basic(t *testing.T) {
	collector := &APIGatewayCollector{
		clientsV1: make(map[string]*apigateway.Client),
		clientsV2: make(map[string]*apigatewayv2.Client),
	}
	assert.Equal(t, "apigateway", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestAPIGatewayCollector_Collect_NoClient(t *testing.T) {
	collector := &APIGatewayCollector{
		clientsV1: make(map[string]*apigateway.Client),
		clientsV2: make(map[string]*apigatewayv2.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no API Gateway v1 client found for region")
}

func TestAPIGatewayCollector_GetColumns(t *testing.T) {
	collector := &APIGatewayCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"Description", "ID", "ProtocolType", "WAF", "AuthorizerType",
		"AuthorizerProviderARN", "CreatedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "API Gateway",
		SubCategory:    "REST API",
		SubSubCategory: "",
		Name:           "test-api",
		Region:         "us-east-1",
		RawData: map[string]interface{}{
			"Description":           "Test API",
			"ID":                    "test-api-id",
			"ProtocolType":          "REST",
			"WAF":                   "test-waf",
			"AuthorizerType":        "JWT",
			"AuthorizerProviderARN": "arn:aws:cognito:us-east-1:123456789012:userpool/us-east-1_abc123",
			"CreatedDate":           "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"API Gateway", "REST API", "", "test-api", "us-east-1",
		"Test API", "test-api-id", "REST", "test-waf", "JWT",
		"arn:aws:cognito:us-east-1:123456789012:userpool/us-east-1_abc123", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
