package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewACMCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewACMCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewACMCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewACMCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestACMCollector_Basic(t *testing.T) {
	collector := &ACMCollector{
		clients: make(map[string]*acm.Client),
	}
	assert.Equal(t, "acm", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestACMCollector_Collect_NoClient(t *testing.T) {
	collector := &ACMCollector{
		clients: make(map[string]*acm.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestACMCollector_GetColumns(t *testing.T) {
	collector := &ACMCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region", "ARN",
		"Type", "KeyAlgorithm", "InUse", "Status", "CreatedDate", "IssuedDate", "ExpirationDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Security",
		SubCategory1: "ACM",
		Name:         "example.com",
		Region:       "us-east-1",
		ARN:          "arn:aws:acm:us-east-1:123456789012:certificate/test-cert",
		RawData: map[string]interface{}{
			"Status":         "ISSUED",
			"Type":           "AMAZON_ISSUED",
			"KeyAlgorithm":   "RSA_2048",
			"InUse":          "test-alb",
			"RequestDate":    "2023-09-25T01:07:55Z",
			"IssuedDate":     "2023-09-25T01:07:55Z",
			"ExpirationDate": "2024-09-25T01:07:55Z",
			"CreatedDate":    "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Security", "ACM", "example.com", "us-east-1", "arn:aws:acm:us-east-1:123456789012:certificate/test-cert",
		"AMAZON_ISSUED", "RSA_2048", "test-alb", "ISSUED", "2023-09-25T01:07:55Z", "2023-09-25T01:07:55Z", "2024-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
