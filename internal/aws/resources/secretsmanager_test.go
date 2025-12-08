package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewSecretsManagerCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewSecretsManagerCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewSecretsManagerCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewSecretsManagerCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestSecretsManagerCollector_Basic(t *testing.T) {
	collector := &SecretsManagerCollector{
		clients: make(map[string]*secretsmanager.Client),
	}
	assert.Equal(t, "secretsmanager", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestSecretsManagerCollector_GetColumns(t *testing.T) {
	collector := &SecretsManagerCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"Description", "KmsKey", "RotationEnabled", "RotationLambdaARN", "LastAccessedDate", "LastRotatedDate", "LastChangedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Security",
		SubCategory:    "SecretsManager",
		SubSubCategory: "Secret",
		Name:           "test-secret",
		Region:         "us-east-1",
		ARN:            "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret-AbCdEf",
		RawData: map[string]interface{}{
			"Description":       "Test secret",
			"KmsKey":            "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			"RotationEnabled":   "true",
			"RotationLambdaARN": "arn:aws:lambda:us-east-1:123456789012:function:rotation-function",
			"LastAccessedDate":  "2023-09-24T01:07:55Z",
			"LastRotatedDate":   "2023-09-25T01:07:55Z",
			"LastChangedDate":   "2023-09-26T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Security", "SecretsManager", "Secret", "test-secret", "us-east-1", "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret-AbCdEf",
		"Test secret", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "true", "arn:aws:lambda:us-east-1:123456789012:function:rotation-function", "2023-09-24T01:07:55Z", "2023-09-25T01:07:55Z", "2023-09-26T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
