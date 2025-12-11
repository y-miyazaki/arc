package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCloudFormationCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewCloudFormationCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewCloudFormationCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewCloudFormationCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestCloudFormationCollector_Basic(t *testing.T) {
	collector := &CloudFormationCollector{
		clients: make(map[string]*cloudformation.Client),
	}
	assert.Equal(t, "cloudformation", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCloudFormationCollector_Collect_NoClient(t *testing.T) {
	collector := &CloudFormationCollector{
		clients: make(map[string]*cloudformation.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestCloudFormationCollector_GetColumns(t *testing.T) {
	collector := &CloudFormationCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region", "ARN",
		"Description", "Type", "Outputs", "Parameters", "Resources", "Status", "DriftStatus", "CreatedDate", "UpdatedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "CloudFormation",
		SubCategory1: "Stack",
		Name:         "test-stack",
		Region:       "us-east-1",
		ARN:          "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/12345678-1234-1234-1234-123456789012",
		RawData: map[string]interface{}{
			"Description": "Test CloudFormation stack",
			"Type":        "Stack",
			"Outputs":     "[]",
			"Parameters":  "{}",
			"Resources":   "5",
			"Status":      "CREATE_COMPLETE",
			"DriftStatus": "IN_SYNC",
			"CreatedDate": "2023-09-25T01:07:55Z",
			"UpdatedDate": "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"CloudFormation", "Stack", "test-stack", "us-east-1", "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/12345678-1234-1234-1234-123456789012",
		"Test CloudFormation stack", "Stack", "[]", "{}", "5", "CREATE_COMPLETE", "IN_SYNC", "2023-09-25T01:07:55Z", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
