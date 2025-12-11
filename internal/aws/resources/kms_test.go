package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewKMSCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewKMSCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewKMSCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewKMSCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestKMSCollector_Basic(t *testing.T) {
	collector := &KMSCollector{
		clients: map[string]*kms.Client{},
	}
	assert.Equal(t, "kms", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestKMSCollector_GetColumns(t *testing.T) {
	collector := &KMSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region",
		"ARN", "Description", "KeyUsage", "KeyManager", "State",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Security",
		SubCategory1: "KMS",
		Name:         "alias/test-key",
		Region:       "us-east-1",
		ARN:          "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
		RawData: map[string]interface{}{
			"Description": "Test KMS key",
			"KeyUsage":    "ENCRYPT_DECRYPT",
			"KeyManager":  "CUSTOMER",
			"State":       "Enabled",
		},
	}

	expectedValues := []string{
		"Security", "KMS", "alias/test-key", "us-east-1",
		"arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "Test KMS key", "ENCRYPT_DECRYPT", "CUSTOMER", "Enabled",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
