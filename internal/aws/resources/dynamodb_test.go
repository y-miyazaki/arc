package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewDynamoDBCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewDynamoDBCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewDynamoDBCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewDynamoDBCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestDynamoDBCollector_Basic(t *testing.T) {
	collector := &DynamoDBCollector{
		clients: make(map[string]*dynamodb.Client),
	}
	assert.Equal(t, "dynamodb", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestDynamoDBCollector_Collect_NoClient(t *testing.T) {
	collector := &DynamoDBCollector{
		clients: make(map[string]*dynamodb.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestDynamoDBCollector_GetColumns(t *testing.T) {
	collector := &DynamoDBCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region", "ARN",
		"AttributeDefinitions", "BillingMode", "StreamEnabled", "GlobalTable", "PointInTimeRecovery",
		"RecoveryPeriodInDays", "EarliestRestorableDateTime", "LatestRestorableDateTime", "DeletionProtection",
		"TTLAttribute", "SSE", "KmsKey", "ItemCount", "TableSize(Bytes)", "Status", "CreationDateTime",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "DynamoDB",
		SubCategory1: "Table",
		Name:         "test-table",
		Region:       "us-east-1",
		ARN:          "arn:aws:dynamodb:us-east-1:123456789012:table/test-table",
		RawData: map[string]interface{}{
			"AttributeDefinitions":       "id:S",
			"BillingMode":                "PAY_PER_REQUEST",
			"StreamEnabled":              "false",
			"GlobalTable":                "false",
			"PointInTimeRecovery":        "ENABLED",
			"RecoveryPeriodInDays":       "35",
			"EarliestRestorableDateTime": "2023-01-01T00:00:00Z",
			"LatestRestorableDateTime":   "2023-12-01T00:00:00Z",
			"DeletionProtection":         "ENABLED",
			"TTLAttribute":               "ttl",
			"SSE":                        "ENABLED",
			"KmsKey":                     "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			"ItemCount":                  "1000",
			"TableSize":                  "1048576",
			"Status":                     "ACTIVE",
			"CreationDateTime":           "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"DynamoDB", "Table", "test-table", "us-east-1", "arn:aws:dynamodb:us-east-1:123456789012:table/test-table",
		"id:S", "PAY_PER_REQUEST", "false", "false", "ENABLED",
		"35", "2023-01-01T00:00:00Z", "2023-12-01T00:00:00Z", "ENABLED",
		"ttl", "ENABLED", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "1000", "1048576", "ACTIVE", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
