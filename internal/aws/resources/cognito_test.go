package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCognitoCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewCognitoCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.idpClients, 2)
	assert.Len(t, collector.identityClients, 2)
	assert.Contains(t, collector.idpClients, "us-east-1")
	assert.Contains(t, collector.idpClients, "eu-west-1")
	assert.Contains(t, collector.identityClients, "us-east-1")
	assert.Contains(t, collector.identityClients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewCognitoCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewCognitoCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.idpClients)
	assert.Empty(t, collector.identityClients)
	assert.NotNil(t, collector.nameResolver)
}

func TestCognitoCollector_Basic(t *testing.T) {
	collector := &CognitoCollector{
		idpClients:      make(map[string]*cognitoidentityprovider.Client),
		identityClients: make(map[string]*cognitoidentity.Client),
	}
	assert.Equal(t, "cognito", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCognitoCollector_Collect_NoClient(t *testing.T) {
	collector := &CognitoCollector{
		idpClients:      make(map[string]*cognitoidentityprovider.Client),
		identityClients: make(map[string]*cognitoidentity.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestCognitoCollector_GetColumns(t *testing.T) {
	collector := &CognitoCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ID",
		"AllowUnauthenticated", "CreationDate", "LastModifiedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Cognito",
		SubCategory:    "User Pool",
		SubSubCategory: "",
		Name:           "test-user-pool",
		Region:         "us-east-1",
		ARN:            "us-east-1_123456789",
		RawData: map[string]interface{}{
			"AllowUnauthenticated": "false",
			"CreationDate":         "2023-09-25T01:07:55Z",
			"LastModifiedDate":     "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Cognito", "User Pool", "", "test-user-pool", "us-east-1", "us-east-1_123456789",
		"false", "2023-09-25T01:07:55Z", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
