package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockSNSCollector is a testable version of SNSCollector that uses mock data
type MockSNSCollector struct{}

func NewMockSNSCollector() *MockSNSCollector {
	return &MockSNSCollector{}
}

func (c *MockSNSCollector) Name() string {
	return "sns"
}

func (c *MockSNSCollector) ShouldSort() bool {
	return true
}

func (c *MockSNSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "DisplayName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DisplayName") }},
		{Header: "Owner", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Owner") }},
		{Header: "Policy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Policy") }},
	}
}

func (c *MockSNSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock SNS Topic
	r1 := Resource{
		Category:    "sns",
		SubCategory: "Topic",
		Name:        "my-notification-topic",
		Region:      region,
		ARN:         "arn:aws:sns:us-east-1:123456789012:my-notification-topic",
		RawData: helpers.NormalizeRawData(map[string]any{
			"DisplayName": "My Notification Topic",
			"Owner":       "123456789012",
			"Policy":      "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"SNS:Publish\",\"Resource\":\"*\"}]}",
		}),
	}
	resources = append(resources, r1)

	return resources, nil
}

func TestSNSCollector_Basic(t *testing.T) {
	collector := &SNSCollector{}
	assert.Equal(t, "sns", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestSNSCollector_GetColumns(t *testing.T) {
	collector := &SNSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "DisplayName", "Owner", "Policy",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:    "sns",
		SubCategory: "Topic",
		Name:        "my-notification-topic",
		Region:      "us-east-1",
		ARN:         "arn:aws:sns:us-east-1:123456789012:my-notification-topic",
		RawData: map[string]any{
			"DisplayName": "My Notification Topic",
			"Owner":       "123456789012",
			"Policy":      "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"SNS:Publish\",\"Resource\":\"*\"}]}",
		},
	}

	expectedValues := []string{
		"sns", "Topic", "", "my-notification-topic", "us-east-1",
		"arn:aws:sns:us-east-1:123456789012:my-notification-topic",
		"My Notification Topic", "123456789012",
		"{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"SNS:Publish\",\"Resource\":\"*\"}]}",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockSNSCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockSNSCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 1)

	// Check resource (Topic)
	r1 := resources[0]
	assert.Equal(t, "sns", r1.Category)
	assert.Equal(t, "Topic", r1.SubCategory)
	assert.Equal(t, "my-notification-topic", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:sns:us-east-1:123456789012:my-notification-topic", r1.ARN)
	assert.Equal(t, "My Notification Topic", helpers.GetMapValue(r1.RawData, "DisplayName"))
	assert.Equal(t, "123456789012", helpers.GetMapValue(r1.RawData, "Owner"))
	assert.Equal(t, "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"SNS:Publish\",\"Resource\":\"*\"}]}", helpers.GetMapValue(r1.RawData, "Policy"))
}
