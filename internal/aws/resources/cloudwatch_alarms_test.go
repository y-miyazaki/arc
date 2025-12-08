package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockCloudWatchAlarmsCollector is a testable version of CloudWatchAlarmsCollector that uses mock data
type MockCloudWatchAlarmsCollector struct{}

func NewMockCloudWatchAlarmsCollector() *MockCloudWatchAlarmsCollector {
	return &MockCloudWatchAlarmsCollector{}
}

func (c *MockCloudWatchAlarmsCollector) Name() string {
	return "cloudwatch_alarms"
}

func (c *MockCloudWatchAlarmsCollector) ShouldSort() bool {
	return true
}

func (c *MockCloudWatchAlarmsCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "MetricName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MetricName") }},
		{Header: "Namespace", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Namespace") }},
		{Header: "Statistic", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Statistic") }},
		{Header: "Threshold", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Threshold") }},
		{Header: "ComparisonOperator", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ComparisonOperator") }},
		{Header: "EvaluationPeriods", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EvaluationPeriods") }},
		{Header: "Period", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Period") }},
		{Header: "TreatMissingData", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TreatMissingData") }},
	}
}

func (c *MockCloudWatchAlarmsCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Metric Alarm
	r1 := Resource{
		Category:    "cloudwatch",
		SubCategory: "Alarm",
		Name:        "HighCPUUtilization",
		Region:      region,
		ARN:         "arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighCPUUtilization",
		RawData: helpers.NormalizeRawData(map[string]any{
			"MetricName":         "CPUUtilization",
			"Namespace":          "AWS/EC2",
			"Statistic":          "Average",
			"Threshold":          80.0,
			"ComparisonOperator": "GreaterThanThreshold",
			"EvaluationPeriods":  2,
			"Period":             300,
			"TreatMissingData":   "missing",
		}),
	}
	resources = append(resources, r1)

	// Mock Composite Alarm
	r2 := Resource{
		Category:    "cloudwatch",
		SubCategory: "Alarm",
		Name:        "CompositeAlarm",
		Region:      region,
		ARN:         "arn:aws:cloudwatch:us-east-1:123456789012:alarm:CompositeAlarm",
		RawData: helpers.NormalizeRawData(map[string]any{
			"MetricName": "Composite",
			"Namespace":  "Composite",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestCloudWatchAlarmsCollector_Basic(t *testing.T) {
	collector := &CloudWatchAlarmsCollector{}
	assert.Equal(t, "cloudwatch_alarms", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCloudWatchAlarmsCollector_GetColumns(t *testing.T) {
	collector := &CloudWatchAlarmsCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "MetricName", "Namespace", "Statistic", "Threshold",
		"ComparisonOperator", "EvaluationPeriods", "Period", "TreatMissingData",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "cloudwatch",
		SubCategory:    "Alarm",
		SubSubCategory: "",
		Name:           "HighCPUUtilization",
		Region:         "us-east-1",
		ARN:            "arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighCPUUtilization",
		RawData: map[string]any{
			"MetricName":         "CPUUtilization",
			"Namespace":          "AWS/EC2",
			"Statistic":          "Average",
			"Threshold":          "80",
			"ComparisonOperator": "GreaterThanThreshold",
			"EvaluationPeriods":  "2",
			"Period":             "300",
			"TreatMissingData":   "missing",
		},
	}

	expectedValues := []string{
		"cloudwatch", "Alarm", "", "HighCPUUtilization", "us-east-1",
		"arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighCPUUtilization",
		"CPUUtilization", "AWS/EC2", "Average", "80", "GreaterThanThreshold",
		"2", "300", "missing",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockCloudWatchAlarmsCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockCloudWatchAlarmsCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (Metric Alarm)
	r1 := resources[0]
	assert.Equal(t, "cloudwatch", r1.Category)
	assert.Equal(t, "Alarm", r1.SubCategory)
	assert.Equal(t, "HighCPUUtilization", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighCPUUtilization", r1.ARN)
	assert.Equal(t, "CPUUtilization", helpers.GetMapValue(r1.RawData, "MetricName"))
	assert.Equal(t, "AWS/EC2", helpers.GetMapValue(r1.RawData, "Namespace"))
	assert.Equal(t, "Average", helpers.GetMapValue(r1.RawData, "Statistic"))
	assert.Equal(t, "80", helpers.GetMapValue(r1.RawData, "Threshold"))
	assert.Equal(t, "GreaterThanThreshold", helpers.GetMapValue(r1.RawData, "ComparisonOperator"))
	assert.Equal(t, "2", helpers.GetMapValue(r1.RawData, "EvaluationPeriods"))
	assert.Equal(t, "300", helpers.GetMapValue(r1.RawData, "Period"))
	assert.Equal(t, "missing", helpers.GetMapValue(r1.RawData, "TreatMissingData"))

	// Check second resource (Composite Alarm)
	r2 := resources[1]
	assert.Equal(t, "cloudwatch", r2.Category)
	assert.Equal(t, "Alarm", r2.SubCategory)
	assert.Equal(t, "CompositeAlarm", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:cloudwatch:us-east-1:123456789012:alarm:CompositeAlarm", r2.ARN)
	assert.Equal(t, "Composite", helpers.GetMapValue(r2.RawData, "MetricName"))
	assert.Equal(t, "Composite", helpers.GetMapValue(r2.RawData, "Namespace"))
}
