package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCognitoIdentityPoolCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewCognitoIdentityPoolCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewCognitoIdentityPoolCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewCognitoIdentityPoolCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestCognitoIdentityPoolCollector_Basic(t *testing.T) {
	collector := &CognitoIdentityPoolCollector{
		clients: make(map[string]*cognitoidentity.Client),
	}
	assert.Equal(t, "cognito_identity_pool", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCognitoIdentityPoolCollector_Collect_NoClient(t *testing.T) {
	collector := &CognitoIdentityPoolCollector{
		clients: make(map[string]*cognitoidentity.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestCognitoIdentityPoolCollector_GetColumns(t *testing.T) {
	collector := &CognitoIdentityPoolCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ID",
		"AllowUnauthenticated", "DeveloperProviderName", "SupportedLoginProviders",
		"CognitoIdentityProviders", "OpenIdConnectProviderARNs", "SamlProviderARNs",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Cognito",
		SubCategory1: "IdentityPool",
		SubCategory2: "",
		Name:         "test-identity-pool",
		Region:       "us-east-1",
		ARN:          "us-east-1:12345678-1234-1234-1234-123456789012",
		RawData: map[string]interface{}{
			"AllowUnauthenticated":      "true",
			"DeveloperProviderName":     "dev-provider",
			"SupportedLoginProviders":   []string{"graph.facebook.com=12345"},
			"CognitoIdentityProviders":  []string{"cognito-idp.us-east-1.amazonaws.com/region=clientid"},
			"OpenIdConnectProviderARNs": []string{"arn:aws:..."},
			"SamlProviderARNs":          []string{"arn:aws:saml:..."},
		},
	}

	expectedValues := []string{
		"Cognito", "IdentityPool", "", "test-identity-pool", "us-east-1", "us-east-1:12345678-1234-1234-1234-123456789012",
		"true", "dev-provider", "graph.facebook.com=12345",
		"cognito-idp.us-east-1.amazonaws.com/region=clientid", "arn:aws:...", "arn:aws:saml:...",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
