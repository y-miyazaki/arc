package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockEventBridgeCollector is a testable version of EventBridgeCollector that uses mock data
type MockEventBridgeCollector struct{}

func NewMockEventBridgeCollector() *MockEventBridgeCollector {
	return &MockEventBridgeCollector{}
}

func (c *MockEventBridgeCollector) Name() string {
	return "eventbridge"
}

func (c *MockEventBridgeCollector) ShouldSort() bool {
	return true
}

func (c *MockEventBridgeCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "ScheduleExpression", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ScheduleExpression") }},
		{Header: "Target", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Target") }},
		{Header: "RetryMaxAttempts", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RetryMaxAttempts") }},
		{Header: "RetryMaxEventAgeSeconds", Value: func(r Resource) string {
			return helpers.GetMapValue(r.RawData, "RetryMaxEventAgeSeconds")
		}},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

func (c *MockEventBridgeCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Rule
	r1 := Resource{
		Category:    "eventbridge",
		SubCategory: "Rule",
		Name:        "MyScheduledRule",
		Region:      region,
		ARN:         "arn:aws:events:us-east-1:123456789012:rule/MyScheduledRule",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description":             "Rule for scheduled tasks",
			"RoleARN":                 "arn:aws:iam::123456789012:role/EventBridgeRole",
			"ScheduleExpression":      "rate(1 hour)",
			"Target":                  "arn:aws:lambda:us-east-1:123456789012:function:MyFunction",
			"RetryMaxAttempts":        "3",
			"RetryMaxEventAgeSeconds": "3600",
			"State":                   "ENABLED",
		}),
	}
	resources = append(resources, r1)

	// Mock Scheduler
	r2 := Resource{
		Category:    "eventbridge",
		SubCategory: "Scheduler",
		Name:        "MySchedule",
		Region:      region,
		ARN:         "arn:aws:scheduler:us-east-1:123456789012:schedule/default/MySchedule",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description":             "Scheduled task using EventBridge Scheduler",
			"RoleARN":                 "arn:aws:iam::123456789012:role/SchedulerRole",
			"ScheduleExpression":      "cron(0 12 * * ? *)",
			"Target":                  "arn:aws:lambda:us-east-1:123456789012:function:MyScheduledFunction",
			"RetryMaxAttempts":        "2",
			"RetryMaxEventAgeSeconds": "1800",
			"State":                   "ENABLED",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestEventBridgeCollector_Basic(t *testing.T) {
	collector := &EventBridgeCollector{}
	assert.Equal(t, "eventbridge", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestEventBridgeCollector_GetColumns(t *testing.T) {
	collector := &EventBridgeCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Description", "RoleARN", "ScheduleExpression", "Target",
		"RetryMaxAttempts", "RetryMaxEventAgeSeconds", "State",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "eventbridge",
		SubCategory:    "Rule",
		SubSubCategory: "",
		Name:           "daily-backup-rule",
		Region:         "us-east-1",
		ARN:            "arn:aws:events:us-east-1:123456789012:rule/daily-backup-rule",
		RawData: map[string]any{
			"Description":             "Daily backup rule",
			"RoleARN":                 "arn:aws:iam::123456789012:role/EventBridgeRole",
			"ScheduleExpression":      "cron(0 2 * * ? *)",
			"Target":                  "arn:aws:lambda:us-east-1:123456789012:function:backup-function",
			"RetryMaxAttempts":        "3",
			"RetryMaxEventAgeSeconds": "3600",
			"State":                   "ENABLED",
		},
	}

	expectedValues := []string{
		"eventbridge", "Rule", "", "daily-backup-rule", "us-east-1",
		"arn:aws:events:us-east-1:123456789012:rule/daily-backup-rule",
		"Daily backup rule", "arn:aws:iam::123456789012:role/EventBridgeRole",
		"cron(0 2 * * ? *)", "arn:aws:lambda:us-east-1:123456789012:function:backup-function",
		"3", "3600", "ENABLED",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockEventBridgeCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockEventBridgeCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (Rule)
	r1 := resources[0]
	assert.Equal(t, "eventbridge", r1.Category)
	assert.Equal(t, "Rule", r1.SubCategory)
	assert.Equal(t, "MyScheduledRule", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:events:us-east-1:123456789012:rule/MyScheduledRule", r1.ARN)
	assert.Equal(t, "Rule for scheduled tasks", helpers.GetMapValue(r1.RawData, "Description"))
	assert.Equal(t, "arn:aws:iam::123456789012:role/EventBridgeRole", helpers.GetMapValue(r1.RawData, "RoleARN"))
	assert.Equal(t, "rate(1 hour)", helpers.GetMapValue(r1.RawData, "ScheduleExpression"))
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:MyFunction", helpers.GetMapValue(r1.RawData, "Target"))
	assert.Equal(t, "3", helpers.GetMapValue(r1.RawData, "RetryMaxAttempts"))
	assert.Equal(t, "3600", helpers.GetMapValue(r1.RawData, "RetryMaxEventAgeSeconds"))
	assert.Equal(t, "ENABLED", helpers.GetMapValue(r1.RawData, "State"))

	// Check second resource (Scheduler)
	r2 := resources[1]
	assert.Equal(t, "eventbridge", r2.Category)
	assert.Equal(t, "Scheduler", r2.SubCategory)
	assert.Equal(t, "MySchedule", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:scheduler:us-east-1:123456789012:schedule/default/MySchedule", r2.ARN)
	assert.Equal(t, "Scheduled task using EventBridge Scheduler", helpers.GetMapValue(r2.RawData, "Description"))
	assert.Equal(t, "arn:aws:iam::123456789012:role/SchedulerRole", helpers.GetMapValue(r2.RawData, "RoleARN"))
	assert.Equal(t, "cron(0 12 * * ? *)", helpers.GetMapValue(r2.RawData, "ScheduleExpression"))
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:MyScheduledFunction", helpers.GetMapValue(r2.RawData, "Target"))
	assert.Equal(t, "2", helpers.GetMapValue(r2.RawData, "RetryMaxAttempts"))
	assert.Equal(t, "1800", helpers.GetMapValue(r2.RawData, "RetryMaxEventAgeSeconds"))
	assert.Equal(t, "ENABLED", helpers.GetMapValue(r2.RawData, "State"))
}
