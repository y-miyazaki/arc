package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCloudWatchLogsCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewCloudWatchLogsCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewCloudWatchLogsCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewCloudWatchLogsCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestCloudWatchLogsCollector_Basic(t *testing.T) {
	collector := &CloudWatchLogsCollector{
		clients: make(map[string]*cloudwatchlogs.Client),
	}
	assert.Equal(t, "cloudwatch_logs", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCloudWatchLogsCollector_Collect_NoClient(t *testing.T) {
	collector := &CloudWatchLogsCollector{
		clients: make(map[string]*cloudwatchlogs.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestCloudWatchLogsCollector_GetColumns(t *testing.T) {
	collector := &CloudWatchLogsCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"RetentionInDays", "StoredBytes", "MetricFilterCount", "SubscriptionFilterCount", "KmsKey", "CreationTime",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "CloudWatch",
		SubCategory:    "Logs",
		SubSubCategory: "Log Group",
		Name:           "test-log-group",
		Region:         "us-east-1",
		ARN:            "arn:aws:logs:us-east-1:123456789012:log-group:test-log-group:*",
		RawData: map[string]interface{}{
			"RetentionInDays":         "30",
			"StoredBytes":             "1024",
			"MetricFilterCount":       "2",
			"SubscriptionFilterCount": "1",
			"KmsKey":                  "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			"CreationTime":            "1695600475",
		},
	}

	expectedValues := []string{
		"CloudWatch", "Logs", "Log Group", "test-log-group", "us-east-1", "arn:aws:logs:us-east-1:123456789012:log-group:test-log-group:*",
		"30", "1024", "2", "1", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "1695600475",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
