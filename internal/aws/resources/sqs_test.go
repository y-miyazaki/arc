package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockSQSCollector is a testable version of SQSCollector that uses mock data
type MockSQSCollector struct{}

func NewMockSQSCollector() *MockSQSCollector {
	return &MockSQSCollector{}
}

func (c *MockSQSCollector) Name() string {
	return "sqs"
}

func (c *MockSQSCollector) ShouldSort() bool {
	return true
}

func (c *MockSQSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "DelaySeconds", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DelaySeconds") }},
		{Header: "MaximumMessageSize", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MaximumMessageSize") }},
		{Header: "MessageRetentionPeriod", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MessageRetentionPeriod") }},
		{Header: "ReceiveMessageWaitTimeSeconds", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ReceiveMessageWaitTimeSeconds") }},
		{Header: "VisibilityTimeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "VisibilityTimeout") }},
		{Header: "RedrivePolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RedrivePolicy") }},
		{Header: "CreatedTimestamp", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedTimestamp") }},
		{Header: "LastModifiedTimestamp", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastModifiedTimestamp") }},
	}
}

func (c *MockSQSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock queue 1
	r1 := Resource{
		Category:    "sqs",
		SubCategory: "Queue",
		Name:        "order-processing-queue",
		Region:      region,
		ARN:         "arn:aws:sqs:us-east-1:123456789012:order-processing-queue",
		RawData: helpers.NormalizeRawData(map[string]any{
			"DelaySeconds":                  "0",
			"MaximumMessageSize":            "262144",
			"MessageRetentionPeriod":        "345600",
			"ReceiveMessageWaitTimeSeconds": "0",
			"VisibilityTimeout":             "30",
			"RedrivePolicy":                 "N/A",
			"CreatedTimestamp":              "2023-08-10T09:00:00Z",
			"LastModifiedTimestamp":         "2023-08-10T09:00:00Z",
		}),
	}
	resources = append(resources, r1)

	// Mock queue 2
	r2 := Resource{
		Category:    "sqs",
		SubCategory: "Queue",
		Name:        "dead-letter-queue",
		Region:      region,
		ARN:         "arn:aws:sqs:us-east-1:123456789012:dead-letter-queue",
		RawData: helpers.NormalizeRawData(map[string]any{
			"DelaySeconds":                  "60",
			"MaximumMessageSize":            "262144",
			"MessageRetentionPeriod":        "1209600",
			"ReceiveMessageWaitTimeSeconds": "20",
			"VisibilityTimeout":             "300",
			"RedrivePolicy":                 "{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:123456789012:order-processing-queue\",\"maxReceiveCount\":5}",
			"CreatedTimestamp":              "2023-07-15T14:30:00Z",
			"LastModifiedTimestamp":         "2023-09-01T11:15:00Z",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestSQSCollector_Basic(t *testing.T) {
	collector := &SQSCollector{}
	assert.Equal(t, "sqs", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestSQSCollector_GetColumns(t *testing.T) {
	collector := &SQSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "DelaySeconds", "MaximumMessageSize", "MessageRetentionPeriod",
		"ReceiveMessageWaitTimeSeconds", "VisibilityTimeout", "RedrivePolicy",
		"CreatedTimestamp", "LastModifiedTimestamp",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "Messaging",
		SubCategory:    "SQS",
		SubSubCategory: "Queue",
		Name:           "my-queue",
		Region:         "us-east-1",
		ARN:            "arn:aws:sqs:us-east-1:123456789012:my-queue",
		RawData: map[string]any{
			"DelaySeconds":                  "30",
			"MaximumMessageSize":            "262144",
			"MessageRetentionPeriod":        "345600",
			"ReceiveMessageWaitTimeSeconds": "0",
			"VisibilityTimeout":             "30",
			"RedrivePolicy":                 "{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:123456789012:dlq\",\"maxReceiveCount\":\"5\"}",
			"CreatedTimestamp":              "1695601655",
			"LastModifiedTimestamp":         "1695601655",
		},
	}

	expectedValues := []string{
		"Messaging", "SQS", "Queue", "my-queue", "us-east-1",
		"arn:aws:sqs:us-east-1:123456789012:my-queue",
		"30", "262144", "345600", "0", "30",
		"{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:123456789012:dlq\",\"maxReceiveCount\":\"5\"}",
		"1695601655", "1695601655",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockSQSCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockSQSCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource
	r1 := resources[0]
	assert.Equal(t, "sqs", r1.Category)
	assert.Equal(t, "Queue", r1.SubCategory)
	assert.Equal(t, "order-processing-queue", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:sqs:us-east-1:123456789012:order-processing-queue", r1.ARN)
	assert.Equal(t, "0", helpers.GetMapValue(r1.RawData, "DelaySeconds"))
	assert.Equal(t, "262144", helpers.GetMapValue(r1.RawData, "MaximumMessageSize"))
	assert.Equal(t, "345600", helpers.GetMapValue(r1.RawData, "MessageRetentionPeriod"))
	assert.Equal(t, "0", helpers.GetMapValue(r1.RawData, "ReceiveMessageWaitTimeSeconds"))
	assert.Equal(t, "30", helpers.GetMapValue(r1.RawData, "VisibilityTimeout"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r1.RawData, "RedrivePolicy"))
	assert.Equal(t, "2023-08-10T09:00:00Z", helpers.GetMapValue(r1.RawData, "CreatedTimestamp"))
	assert.Equal(t, "2023-08-10T09:00:00Z", helpers.GetMapValue(r1.RawData, "LastModifiedTimestamp"))

	// Check second resource
	r2 := resources[1]
	assert.Equal(t, "sqs", r2.Category)
	assert.Equal(t, "Queue", r2.SubCategory)
	assert.Equal(t, "dead-letter-queue", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:sqs:us-east-1:123456789012:dead-letter-queue", r2.ARN)
	assert.Equal(t, "60", helpers.GetMapValue(r2.RawData, "DelaySeconds"))
	assert.Equal(t, "262144", helpers.GetMapValue(r2.RawData, "MaximumMessageSize"))
	assert.Equal(t, "1209600", helpers.GetMapValue(r2.RawData, "MessageRetentionPeriod"))
	assert.Equal(t, "20", helpers.GetMapValue(r2.RawData, "ReceiveMessageWaitTimeSeconds"))
	assert.Equal(t, "300", helpers.GetMapValue(r2.RawData, "VisibilityTimeout"))
	assert.Equal(t, "{\"deadLetterTargetArn\":\"arn:aws:sqs:us-east-1:123456789012:order-processing-queue\",\"maxReceiveCount\":5}", helpers.GetMapValue(r2.RawData, "RedrivePolicy"))
	assert.Equal(t, "2023-07-15T14:30:00Z", helpers.GetMapValue(r2.RawData, "CreatedTimestamp"))
	assert.Equal(t, "2023-09-01T11:15:00Z", helpers.GetMapValue(r2.RawData, "LastModifiedTimestamp"))
}

// ParseTimestamp tests moved to helpers package (helpers_test.go).
