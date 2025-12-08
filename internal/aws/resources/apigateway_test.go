package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestAPIGatewayCollector_Name(t *testing.T) {
	collector := &APIGatewayCollector{}
	assert.Equal(t, "apigateway", collector.Name())
}

func TestAPIGatewayCollector_ShouldSort(t *testing.T) {
	collector := &APIGatewayCollector{}
	assert.False(t, collector.ShouldSort())
}

func TestAPIGatewayCollector_GetColumns(t *testing.T) {
	collector := &APIGatewayCollector{}
	columns := collector.GetColumns()

	assert.NotEmpty(t, columns)
	assert.Contains(t, columns[0].Header, "Category")
	assert.Contains(t, columns[1].Header, "SubCategory")
	assert.Contains(t, columns[2].Header, "SubSubCategory")
	assert.Contains(t, columns[3].Header, "Name")
	assert.Contains(t, columns[4].Header, "Region")
	assert.Contains(t, columns[5].Header, "Description")
	assert.Contains(t, columns[6].Header, "ID")
	assert.Contains(t, columns[7].Header, "ProtocolType")

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "apigateway",
		SubCategory:    "REST",
		SubSubCategory: "API",
		Name:           "my-api",
		Region:         "us-east-1",
		RawData: map[string]any{
			"Description":           "Test API description",
			"ID":                    "abc123def4",
			"ProtocolType":          "HTTP",
			"WAF":                   "waf-12345678",
			"AuthorizerType":        "COGNITO_USER_POOLS",
			"AuthorizerProviderARN": "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_ABC123DEF",
		},
	}

	// Test each Value function
	assert.Equal(t, "apigateway", columns[0].Value(sampleResource))
	assert.Equal(t, "REST", columns[1].Value(sampleResource))
	assert.Equal(t, "API", columns[2].Value(sampleResource))
	assert.Equal(t, "my-api", columns[3].Value(sampleResource))
	assert.Equal(t, "us-east-1", columns[4].Value(sampleResource))
	assert.Equal(t, "Test API description", columns[5].Value(sampleResource))
	assert.Equal(t, "abc123def4", columns[6].Value(sampleResource))
	assert.Equal(t, "HTTP", columns[7].Value(sampleResource))
	assert.Equal(t, "waf-12345678", columns[8].Value(sampleResource))
	assert.Equal(t, "COGNITO_USER_POOLS", columns[9].Value(sampleResource))
	assert.Equal(t, "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_ABC123DEF", columns[10].Value(sampleResource))
}

// MockAPIGatewayCollector is a mock implementation of APIGatewayCollector for testing
type MockAPIGatewayCollector struct{}

func (m *MockAPIGatewayCollector) Name() string {
	return "apigateway"
}

func (m *MockAPIGatewayCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	return []Resource{
		{
			Category:    "apigateway",
			SubCategory: "REST",
			Name:        "test-api",
			Region:      region,
			RawData: map[string]any{
				"ID":                    "test-api-id",
				"ProtocolType":          "REST",
				"WAF":                   "waf-id",
				"AuthorizerType":        "COGNITO_USER_POOLS",
				"AuthorizerProviderARN": "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_test",
			},
		},
		{
			Category:    "apigateway",
			SubCategory: "HTTP",
			Name:        "test-http-api",
			Region:      region,
			RawData: map[string]any{
				"ID":           "test-http-api-id",
				"ProtocolType": "HTTP",
				"WAF":          "",
			},
		},
	}, nil
}

func (m *MockAPIGatewayCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "ProtocolType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ProtocolType") }},
	}
}

func (m *MockAPIGatewayCollector) ShouldSort() bool {
	return false
}

func TestMockAPIGatewayCollector_Collect(t *testing.T) {
	collector := &MockAPIGatewayCollector{}
	cfg := &aws.Config{}
	region := "us-east-1"

	resources, err := collector.Collect(context.Background(), cfg, region)

	assert.NoError(t, err)
	assert.NotEmpty(t, resources)
	assert.Equal(t, 2, len(resources))

	// Check REST API resource
	restResource := resources[0]
	assert.Equal(t, "apigateway", restResource.Category)
	assert.Equal(t, "REST", restResource.SubCategory)
	assert.Equal(t, "test-api", restResource.Name)
	assert.Equal(t, region, restResource.Region)

	// Check HTTP API resource
	httpResource := resources[1]
	assert.Equal(t, "apigateway", httpResource.Category)
	assert.Equal(t, "HTTP", httpResource.SubCategory)
	assert.Equal(t, "test-http-api", httpResource.Name)
	assert.Equal(t, region, httpResource.Region)
}
