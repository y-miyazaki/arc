package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCloudWatchAlarmsCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewCloudWatchAlarmsCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewCloudWatchAlarmsCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewCloudWatchAlarmsCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestCloudWatchAlarmsCollector_Basic(t *testing.T) {
	collector := &CloudWatchAlarmsCollector{
		clients: make(map[string]*cloudwatch.Client),
	}
	assert.Equal(t, "cloudwatch_alarms", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCloudWatchAlarmsCollector_Collect_NoClient(t *testing.T) {
	collector := &CloudWatchAlarmsCollector{
		clients: make(map[string]*cloudwatch.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestCloudWatchAlarmsCollector_GetColumns(t *testing.T) {
	collector := &CloudWatchAlarmsCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"MetricName", "Namespace", "Statistic", "Threshold", "ComparisonOperator", "EvaluationPeriods", "Period", "TreatMissingData",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "CloudWatch",
		SubCategory:    "Alarms",
		SubSubCategory: "Metric Alarm",
		Name:           "test-alarm",
		Region:         "us-east-1",
		ARN:            "arn:aws:cloudwatch:us-east-1:123456789012:alarm:test-alarm",
		RawData: map[string]interface{}{
			"MetricName":         "CPUUtilization",
			"Namespace":          "AWS/EC2",
			"Statistic":          "Average",
			"Threshold":          "80.0",
			"ComparisonOperator": "GreaterThanThreshold",
			"EvaluationPeriods":  "2",
			"Period":             "300",
			"TreatMissingData":   "missing",
		},
	}

	expectedValues := []string{
		"CloudWatch", "Metric Alarm", "Metric Alarm", "test-alarm", "us-east-1", "arn:aws:cloudwatch:us-east-1:123456789012:alarm:test-alarm",
		"CPUUtilization", "AWS/EC2", "Average", "80.0", "GreaterThanThreshold", "2", "300", "missing",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
