package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewSNSCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewSNSCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewSNSCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewSNSCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestSNSCollector_Basic(t *testing.T) {
	collector := &SNSCollector{
		clients: make(map[string]*sns.Client),
	}
	assert.Equal(t, "sns", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestSNSCollector_GetColumns(t *testing.T) {
	collector := &SNSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region", "ARN",
		"DisplayName", "Owner", "Policy",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "sns",
		SubCategory1: "Topic",
		Name:         "test-topic",
		Region:       "us-east-1",
		ARN:          "arn:aws:sns:us-east-1:123456789012:test-topic",
		RawData: map[string]interface{}{
			"DisplayName": "Test Topic",
			"Owner":       "123456789012",
			"Policy":      "{\"Version\":\"2012-10-17\"}",
		},
	}

	expectedValues := []string{
		"sns", "Topic", "test-topic", "us-east-1", "arn:aws:sns:us-east-1:123456789012:test-topic",
		"Test Topic", "123456789012", "{\"Version\":\"2012-10-17\"}",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}

func TestSNSCollector_Collect_NoClient(t *testing.T) {
	collector := &SNSCollector{clients: map[string]*sns.Client{}}

	_, err := collector.Collect(context.Background(), "us-west-2")

	require.Error(t, err)
	assert.ErrorContains(t, err, "no client found for region")
}

func TestSNSCollector_Collect_ListTopicsError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &SNSCollector{
		clients: map[string]*sns.Client{
			"us-east-1": sns.NewFromConfig(cfg),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list SNS topics")
}
