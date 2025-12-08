package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockKinesisCollector is a testable version of KinesisCollector that uses mock data
type MockKinesisCollector struct{}

func NewMockKinesisCollector() *MockKinesisCollector {
	return &MockKinesisCollector{}
}

func (c *MockKinesisCollector) Name() string {
	return "kinesis"
}

func (c *MockKinesisCollector) ShouldSort() bool {
	return true
}

func (c *MockKinesisCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "Shards", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Shards") }},
		{Header: "DestinationId", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DestinationId") }},
		{Header: "RetentionPeriodHours", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RetentionPeriodHours") }},
		{Header: "EncryptionType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EncryptionType") }},
		{Header: "CreatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedDate") }},
		{Header: "LastUpdatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastUpdatedDate") }},
	}
}

func (c *MockKinesisCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Stream
	r1 := Resource{
		Category:    "kinesis",
		SubCategory: "Stream",
		Name:        "my-kinesis-stream",
		Region:      region,
		ARN:         "arn:aws:kinesis:us-east-1:123456789012:stream/my-kinesis-stream",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Status":               "ACTIVE",
			"Shards":               2,
			"RetentionPeriodHours": 24,
			"EncryptionType":       "KMS",
			"CreatedDate":          time.Date(2023, 8, 15, 10, 30, 0, 0, time.UTC),
		}),
	}
	resources = append(resources, r1)

	// Mock Firehose Delivery Stream
	r2 := Resource{
		Category:    "kinesis",
		SubCategory: "Firehose",
		Name:        "my-delivery-stream",
		Region:      region,
		ARN:         "arn:aws:firehose:us-east-1:123456789012:deliverystream/my-delivery-stream",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Status":          "ACTIVE",
			"DestinationId":   "S3Destination",
			"CreatedDate":     time.Date(2023, 9, 1, 14, 20, 0, 0, time.UTC),
			"LastUpdatedDate": time.Date(2023, 9, 5, 16, 45, 0, 0, time.UTC),
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestKinesisCollector_Basic(t *testing.T) {
	collector := &KinesisCollector{}
	assert.Equal(t, "kinesis", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestKinesisCollector_GetColumns(t *testing.T) {
	collector := &KinesisCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Status", "Shards", "DestinationId", "RetentionPeriodHours",
		"EncryptionType", "CreatedDate", "LastUpdatedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "kinesis",
		SubCategory:    "Stream",
		SubSubCategory: "",
		Name:           "data-stream",
		Region:         "us-east-1",
		ARN:            "arn:aws:kinesis:us-east-1:123456789012:stream/data-stream",
		RawData: map[string]any{
			"Status":               "ACTIVE",
			"Shards":               "4",
			"DestinationId":        "",
			"RetentionPeriodHours": "168",
			"EncryptionType":       "KMS",
			"CreatedDate":          "2023-08-15T10:30:00Z",
			"LastUpdatedDate":      "2023-08-20T14:45:00Z",
		},
	}

	expectedValues := []string{
		"kinesis", "Stream", "", "data-stream", "us-east-1",
		"arn:aws:kinesis:us-east-1:123456789012:stream/data-stream",
		"ACTIVE", "4", "", "168", "KMS", "2023-08-15T10:30:00Z", "2023-08-20T14:45:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockKinesisCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockKinesisCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (Stream)
	r1 := resources[0]
	assert.Equal(t, "kinesis", r1.Category)
	assert.Equal(t, "Stream", r1.SubCategory)
	assert.Equal(t, "my-kinesis-stream", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:kinesis:us-east-1:123456789012:stream/my-kinesis-stream", r1.ARN)
	assert.Equal(t, "ACTIVE", helpers.GetMapValue(r1.RawData, "Status"))
	assert.Equal(t, "2", helpers.GetMapValue(r1.RawData, "Shards"))
	assert.Equal(t, "24", helpers.GetMapValue(r1.RawData, "RetentionPeriodHours"))
	assert.Equal(t, "KMS", helpers.GetMapValue(r1.RawData, "EncryptionType"))
	assert.Equal(t, "2023-08-15T10:30:00Z", helpers.GetMapValue(r1.RawData, "CreatedDate"))

	// Check second resource (Firehose)
	r2 := resources[1]
	assert.Equal(t, "kinesis", r2.Category)
	assert.Equal(t, "Firehose", r2.SubCategory)
	assert.Equal(t, "my-delivery-stream", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:firehose:us-east-1:123456789012:deliverystream/my-delivery-stream", r2.ARN)
	assert.Equal(t, "ACTIVE", helpers.GetMapValue(r2.RawData, "Status"))
	assert.Equal(t, "S3Destination", helpers.GetMapValue(r2.RawData, "DestinationId"))
	assert.Equal(t, "2023-09-01T14:20:00Z", helpers.GetMapValue(r2.RawData, "CreatedDate"))
	assert.Equal(t, "2023-09-05T16:45:00Z", helpers.GetMapValue(r2.RawData, "LastUpdatedDate"))
}
