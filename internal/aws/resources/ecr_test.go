package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewECRCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewECRCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewECRCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewECRCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestECRCollector_Basic(t *testing.T) {
	collector := &ECRCollector{
		clients: map[string]*ecr.Client{},
	}
	assert.Equal(t, "ecr", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestECRCollector_GetColumns(t *testing.T) {
	collector := &ECRCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region",
		"URI", "Mutability", "Encryption", "KMSKey", "ScanOnPush",
		"LifecyclePolicy", "ImageCount", "CreatedAt",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "ECR",
		SubCategory1: "Repository",
		Name:         "test-repo",
		Region:       "us-east-1",
		RawData: map[string]interface{}{
			"URI":             "123456789012.dkr.ecr.us-east-1.amazonaws.com/test-repo",
			"Mutability":      "MUTABLE",
			"Encryption":      "KMS",
			"KMSKey":          "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			"ScanOnPush":      "true",
			"LifecyclePolicy": "Yes",
			"ImageCount":      "5",
			"CreatedAt":       "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"ECR", "Repository", "test-repo", "us-east-1",
		"123456789012.dkr.ecr.us-east-1.amazonaws.com/test-repo", "MUTABLE", "KMS", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "true",
		"Yes", "5", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
