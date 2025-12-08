package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewEventBridgeCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewEventBridgeCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.ebClients)
	assert.Len(t, collector.ebClients, len(regions))
	assert.NotNil(t, collector.schClients)
	assert.Len(t, collector.schClients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewEventBridgeCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewEventBridgeCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.ebClients)
	assert.Len(t, collector.ebClients, 0)
	assert.NotNil(t, collector.schClients)
	assert.Len(t, collector.schClients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestEventBridgeCollector_Basic(t *testing.T) {
	collector := &EventBridgeCollector{
		ebClients:  map[string]*eventbridge.Client{},
		schClients: map[string]*scheduler.Client{},
	}
	assert.Equal(t, "eventbridge", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestEventBridgeCollector_GetColumns(t *testing.T) {
	collector := &EventBridgeCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"Description", "RoleARN", "ScheduleExpression", "Target", "RetryMaxAttempts",
		"RetryMaxEventAgeSeconds", "State",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "EventBridge",
		SubCategory:    "Rule",
		SubSubCategory: "",
		Name:           "test-rule",
		Region:         "us-east-1",
		ARN:            "arn:aws:events:us-east-1:123456789012:rule/test-rule",
		RawData: map[string]interface{}{
			"Description":             "Test EventBridge rule",
			"RoleARN":                 "arn:aws:iam::123456789012:role/EventBridgeRole",
			"ScheduleExpression":      "rate(1 hour)",
			"Target":                  "arn:aws:lambda:us-east-1:123456789012:function:MyFunction",
			"RetryMaxAttempts":        "3",
			"RetryMaxEventAgeSeconds": "3600",
			"State":                   "ENABLED",
		},
	}

	expectedValues := []string{
		"EventBridge", "Rule", "", "test-rule", "us-east-1", "arn:aws:events:us-east-1:123456789012:rule/test-rule",
		"Test EventBridge rule", "arn:aws:iam::123456789012:role/EventBridgeRole", "rate(1 hour)", "arn:aws:lambda:us-east-1:123456789012:function:MyFunction", "3",
		"3600", "ENABLED",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
