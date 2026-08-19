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
	t.Parallel()

	cfg := &aws.Config{
		Region: "us-east-1",
	}

	tests := []struct {
		name    string
		regions []string
		wantLen int
	}{
		{name: "creates clients for each region", regions: []string{"us-east-1", "eu-west-1"}, wantLen: 2},
		{name: "empty regions", regions: []string{}, wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nameResolver, err := helpers.NewNameResolver(cfg, tt.regions)
			require.NoError(t, err)

			collector, err := NewAPIGatewayCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.Len(t, collector.clientsV1, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.clientsV1, region)
			}
			assert.Len(t, collector.clientsV2, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.clientsV2, region)
			}
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestAPIGatewayCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "apigateway", wantSort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &APIGatewayCollector{
				clientsV1: make(map[string]*apigateway.Client),
				clientsV2: make(map[string]*apigatewayv2.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestAPIGatewayCollector_Collect_NoClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		region       string
		wantContains string
	}{
		{name: "missing client returns error", region: "us-west-2", wantContains: "no API Gateway v1 client found for region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &APIGatewayCollector{
				clientsV1: make(map[string]*apigateway.Client),
				clientsV2: make(map[string]*apigatewayv2.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestAPIGatewayCollector_GetColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		resource    Resource
		wantHeaders []string
		wantValues  []string
	}{
		{
			name: "headers and sample values",
			resource: Resource{
				Category:     "API Gateway",
				SubCategory1: "REST API",
				SubCategory2: "",
				Name:         "test-api",
				Region:       "us-east-1",
				RawData: map[string]any{
					"Description":           "Test API",
					"ID":                    "test-api-id",
					"ProtocolType":          "REST",
					"WAF":                   "test-waf",
					"AuthorizerType":        "JWT",
					"AuthorizerProviderARN": "arn:aws:cognito:us-east-1:123456789012:userpool/us-east-1_abc123",
					"CreatedDate":           "2023-09-25T01:07:55Z",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "SubCategory2", "Name", "Region",
				"Description", "ID", "ProtocolType", "WAF", "AuthorizerType",
				"AuthorizerProviderARN", "CreatedDate",
			},
			wantValues: []string{
				"API Gateway", "REST API", "", "test-api", "us-east-1",
				"Test API", "test-api-id", "REST", "test-waf", "JWT",
				"arn:aws:cognito:us-east-1:123456789012:userpool/us-east-1_abc123", "2023-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &APIGatewayCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
