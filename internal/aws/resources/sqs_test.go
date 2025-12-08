package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewSQSCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewSQSCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewSQSCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewSQSCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestSQSCollector_Basic(t *testing.T) {
	collector := &SQSCollector{
		clients: make(map[string]*sqs.Client),
	}
	assert.Equal(t, "sqs", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestSQSCollector_Collect_NoClient(t *testing.T) {
	collector := &SQSCollector{
		clients: make(map[string]*sqs.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no SQS client found for region")
}

func TestSQSCollector_GetColumns(t *testing.T) {
	collector := &SQSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"DelaySeconds", "MaximumMessageSize", "MessageRetentionPeriod", "ReceiveMessageWaitTimeSeconds",
		"VisibilityTimeout", "RedrivePolicy", "CreatedTimestamp", "LastModifiedTimestamp",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "SQS",
		SubCategory:    "Queue",
		SubSubCategory: "",
		Name:           "test-queue",
		Region:         "us-east-1",
		ARN:            "arn:aws:sqs:us-east-1:123456789012:test-queue",
		RawData: map[string]interface{}{
			"DelaySeconds":                  "0",
			"MaximumMessageSize":            "262144",
			"MessageRetentionPeriod":        "345600",
			"ReceiveMessageWaitTimeSeconds": "0",
			"VisibilityTimeout":             "30",
			"RedrivePolicy":                 "{}",
			"CreatedTimestamp":              "1695600475",
			"LastModifiedTimestamp":         "1695600475",
		},
	}

	expectedValues := []string{
		"SQS", "Queue", "", "test-queue", "us-east-1", "arn:aws:sqs:us-east-1:123456789012:test-queue",
		"0", "262144", "345600", "0",
		"30", "{}", "1695600475", "1695600475",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
