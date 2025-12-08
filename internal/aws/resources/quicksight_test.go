package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/quicksight"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewQuickSightCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewQuickSightCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.stsClient)
	assert.NotNil(t, collector.nameResolver)
}

func TestNewQuickSightCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewQuickSightCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.stsClient)
	assert.NotNil(t, collector.nameResolver)
}

func TestQuickSightCollector_Basic(t *testing.T) {
	collector := &QuickSightCollector{
		clients:   map[string]*quicksight.Client{},
		stsClient: &sts.Client{},
	}
	assert.Equal(t, "quicksight", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestQuickSightCollector_GetColumns(t *testing.T) {
	collector := &QuickSightCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "Type", "Status", "CreatedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Analytics",
		SubCategory:    "QuickSight",
		SubSubCategory: "DataSource",
		Name:           "test-datasource",
		Region:         "us-east-1",
		ARN:            "test-datasource-id",
		RawData: map[string]interface{}{
			"Type":        "REDSHIFT",
			"Status":      "CREATION_SUCCESSFUL",
			"CreatedDate": "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Analytics", "QuickSight", "DataSource", "test-datasource", "us-east-1",
		"test-datasource-id", "REDSHIFT", "CREATION_SUCCESSFUL", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
