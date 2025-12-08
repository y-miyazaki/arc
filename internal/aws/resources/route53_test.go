package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewRoute53Collector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewRoute53Collector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestNewRoute53Collector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewRoute53Collector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestRoute53Collector_Basic(t *testing.T) {
	collector := &Route53Collector{
		client: &route53.Client{},
	}
	assert.Equal(t, "route53", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestRoute53Collector_GetColumns(t *testing.T) {
	collector := &Route53Collector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ID",
		"Type", "Comment", "TTL", "RecordType", "Value", "RecordCount",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Networking",
		SubCategory:    "Route53",
		SubSubCategory: "Record",
		Name:           "example.com",
		Region:         "us-east-1",
		RawData: map[string]interface{}{
			"ID":          "Z123456789",
			"Type":        "Hosted Zone",
			"Comment":     "Test zone",
			"TTL":         "300",
			"RecordType":  "A",
			"Value":       "192.168.1.1",
			"RecordCount": "1",
		},
	}

	expectedValues := []string{
		"Networking", "Route53", "Record", "example.com", "us-east-1", "Z123456789",
		"Hosted Zone", "Test zone", "300", "A", "192.168.1.1", "1",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
