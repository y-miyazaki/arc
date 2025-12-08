package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestS3Collector_Basic(t *testing.T) {
	collector := &S3Collector{}
	assert.Equal(t, "s3", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestS3Collector_GetColumns(t *testing.T) {
	collector := &S3Collector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Encryption", "Versioning", "PABBlockPublicACLs", "PABIgnorePublicACLs",
		"PABBlockPublicPolicy", "PABRestrictPublicBuckets", "AccessLogARN", "LifecycleRules", "CreationDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Storage",
		SubCategory:    "S3",
		SubSubCategory: "Bucket",
		Name:           "my-test-bucket",
		Region:         "us-east-1",
		ARN:            "arn:aws:s3:::my-test-bucket",
		RawData: map[string]interface{}{
			"Encryption":               "AES256",
			"Versioning":               "Enabled",
			"PABBlockPublicACLs":       "true",
			"PABIgnorePublicACLs":      "true",
			"PABBlockPublicPolicy":     "true",
			"PABRestrictPublicBuckets": "true",
			"AccessLogARN":             "arn:aws:s3:::logs-bucket/access-logs/",
			"LifecycleRules":           "1 rule",
			"CreationDate":             "2023-01-01T00:00:00Z",
		},
	}

	expectedValues := []string{
		"Storage", "S3", "Bucket", "my-test-bucket", "us-east-1",
		"arn:aws:s3:::my-test-bucket", "AES256", "Enabled", "true", "true",
		"true", "true", "arn:aws:s3:::logs-bucket/access-logs/", "1 rule", "2023-01-01T00:00:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}

// MockS3Collector is a testable version of S3Collector that uses mock data
type MockS3Collector struct{}

func NewMockS3Collector() *MockS3Collector {
	return &MockS3Collector{}
}

func (c *MockS3Collector) Name() string {
	return "s3"
}

func (c *MockS3Collector) ShouldSort() bool {
	return true
}

func (c *MockS3Collector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Encryption", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Encryption") }},
		{Header: "Versioning", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Versioning") }},
		{Header: "PABBlockPublicACLs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABBlockPublicACLs") }},
		{Header: "PABIgnorePublicACLs", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABIgnorePublicACLs") }},
		{Header: "PABBlockPublicPolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABBlockPublicPolicy") }},
		{Header: "PABRestrictPublicBuckets", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PABRestrictPublicBuckets") }},
		{Header: "AccessLogARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AccessLogARN") }},
		{Header: "LifecycleRules", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LifecycleRules") }},
		{Header: "CreationDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationDate") }},
	}
}

func (c *MockS3Collector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock bucket 1
	r1 := Resource{
		Category:    "s3",
		SubCategory: "Bucket",
		Name:        "my-app-bucket",
		Region:      region,
		ARN:         "arn:aws:s3:::my-app-bucket",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Encryption":               "AES256",
			"Versioning":               "Enabled",
			"PABBlockPublicACLs":       "true",
			"PABIgnorePublicACLs":      "true",
			"PABBlockPublicPolicy":     "true",
			"PABRestrictPublicBuckets": "true",
			"AccessLogARN":             "N/A",
			"LifecycleRules":           "1",
			"CreationDate":             "2023-08-15T10:30:00Z",
		}),
	}
	resources = append(resources, r1)

	// Mock bucket 2
	r2 := Resource{
		Category:    "s3",
		SubCategory: "Bucket",
		Name:        "logs-bucket",
		Region:      region,
		ARN:         "arn:aws:s3:::logs-bucket",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Encryption":               "KMS",
			"Versioning":               "Suspended",
			"PABBlockPublicACLs":       "false",
			"PABIgnorePublicACLs":      "false",
			"PABBlockPublicPolicy":     "false",
			"PABRestrictPublicBuckets": "false",
			"AccessLogARN":             "arn:aws:s3:::access-logs-bucket",
			"LifecycleRules":           "2",
			"CreationDate":             "2023-07-01T09:15:00Z",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestMockS3Collector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockS3Collector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource
	r1 := resources[0]
	assert.Equal(t, "s3", r1.Category)
	assert.Equal(t, "Bucket", r1.SubCategory)
	assert.Equal(t, "my-app-bucket", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:s3:::my-app-bucket", r1.ARN)
	assert.Equal(t, "AES256", helpers.GetMapValue(r1.RawData, "Encryption"))
	assert.Equal(t, "Enabled", helpers.GetMapValue(r1.RawData, "Versioning"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "PABBlockPublicACLs"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "PABIgnorePublicACLs"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "PABBlockPublicPolicy"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "PABRestrictPublicBuckets"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r1.RawData, "AccessLogARN"))
	assert.Equal(t, "1", helpers.GetMapValue(r1.RawData, "LifecycleRules"))
	assert.Equal(t, "2023-08-15T10:30:00Z", helpers.GetMapValue(r1.RawData, "CreationDate"))

	// Check second resource
	r2 := resources[1]
	assert.Equal(t, "s3", r2.Category)
	assert.Equal(t, "Bucket", r2.SubCategory)
	assert.Equal(t, "logs-bucket", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:s3:::logs-bucket", r2.ARN)
	assert.Equal(t, "KMS", helpers.GetMapValue(r2.RawData, "Encryption"))
	assert.Equal(t, "Suspended", helpers.GetMapValue(r2.RawData, "Versioning"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "PABBlockPublicACLs"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "PABIgnorePublicACLs"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "PABBlockPublicPolicy"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "PABRestrictPublicBuckets"))
	assert.Equal(t, "arn:aws:s3:::access-logs-bucket", helpers.GetMapValue(r2.RawData, "AccessLogARN"))
	assert.Equal(t, "2", helpers.GetMapValue(r2.RawData, "LifecycleRules"))
	assert.Equal(t, "2023-07-01T09:15:00Z", helpers.GetMapValue(r2.RawData, "CreationDate"))
}
