package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestECRCollector_Basic(t *testing.T) {
	collector := &ECRCollector{}
	assert.Equal(t, "ecr", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestECRCollector_GetColumns(t *testing.T) {
	collector := &ECRCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"URI", "Mutability", "Encryption", "KMSKey", "ScanOnPush", "LifecyclePolicy", "ImageCount", "CreatedAt",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "ecr",
		SubCategory:    "Repository",
		SubSubCategory: "",
		Name:           "my-app",
		Region:         "us-east-1",
		RawData: map[string]any{
			"URI":             "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			"Mutability":      "MUTABLE",
			"Encryption":      "AES256",
			"KMSKey":          "alias/my-key",
			"ScanOnPush":      "true",
			"LifecyclePolicy": `[{"rulePriority":1,"description":"Expire old images","selection":{"tagStatus":"untagged","countType":"sinceImagePushed","countUnit":"days","countNumber":30},"action":{"type":"expire"}}]`,
			"ImageCount":      "5",
			"CreatedAt":       "2023-01-15T10:30:00Z",
		},
	}

	// Test each Value function
	assert.Equal(t, "ecr", columns[0].Value(sampleResource))
	assert.Equal(t, "Repository", columns[1].Value(sampleResource))
	assert.Equal(t, "", columns[2].Value(sampleResource))
	assert.Equal(t, "my-app", columns[3].Value(sampleResource))
	assert.Equal(t, "us-east-1", columns[4].Value(sampleResource))
	assert.Equal(t, "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app", columns[5].Value(sampleResource))
	assert.Equal(t, "MUTABLE", columns[6].Value(sampleResource))
	assert.Equal(t, "AES256", columns[7].Value(sampleResource))
	assert.Equal(t, "alias/my-key", columns[8].Value(sampleResource))
	assert.Equal(t, "true", columns[9].Value(sampleResource))
	assert.Equal(t, `[{"rulePriority":1,"description":"Expire old images","selection":{"tagStatus":"untagged","countType":"sinceImagePushed","countUnit":"days","countNumber":30},"action":{"type":"expire"}}]`, columns[10].Value(sampleResource))
	assert.Equal(t, "5", columns[11].Value(sampleResource))
	assert.Equal(t, "2023-01-15T10:30:00Z", columns[12].Value(sampleResource))
}

// MockECRCollector is a testable version of ECRCollector that uses mock data
type MockECRCollector struct{}

func NewMockECRCollector() *MockECRCollector {
	return &MockECRCollector{}
}

func (c *MockECRCollector) Name() string {
	return "ecr"
}

func (c *MockECRCollector) ShouldSort() bool {
	return true
}

func (c *MockECRCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "URI", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "URI") }},
		{Header: "Mutability", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Mutability") }},
		{Header: "Encryption", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Encryption") }},
		{Header: "KMSKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KMSKey") }},
		{Header: "ScanOnPush", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ScanOnPush") }},
		{Header: "LifecyclePolicy", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LifecyclePolicy") }},
		{Header: "ImageCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ImageCount") }},
		{Header: "CreatedAt", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreatedAt") }},
	}
}

func (c *MockECRCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock repository 1
	r1 := Resource{
		Category:    "ecr",
		SubCategory: "Repository",
		Name:        "my-app",
		Region:      region,
		ARN:         "arn:aws:ecr:us-east-1:123456789012:repository/my-app",
		RawData: helpers.NormalizeRawData(map[string]any{
			"URI":             "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			"Mutability":      "MUTABLE",
			"Encryption":      "AES256",
			"KMSKey":          "alias/my-key",
			"ScanOnPush":      "true",
			"LifecyclePolicy": `[{"rulePriority":1,"description":"Expire old images","selection":{"tagStatus":"untagged","countType":"sinceImagePushed","countUnit":"days","countNumber":30},"action":{"type":"expire"}}]`,
			"ImageCount":      "5",
			"CreatedAt":       "2023-08-15T10:30:00Z",
		}),
	}
	resources = append(resources, r1)

	// Mock repository 2
	r2 := Resource{
		Category:    "ecr",
		SubCategory: "Repository",
		Name:        "web-service",
		Region:      region,
		ARN:         "arn:aws:ecr:us-east-1:123456789012:repository/web-service",
		RawData: helpers.NormalizeRawData(map[string]any{
			"URI":             "123456789012.dkr.ecr.us-east-1.amazonaws.com/web-service",
			"Mutability":      "IMMUTABLE",
			"Encryption":      "KMS",
			"KMSKey":          "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			"ScanOnPush":      "false",
			"LifecyclePolicy": "",
			"ImageCount":      "12",
			"CreatedAt":       "2023-09-01T14:20:00Z",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestMockECRCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockECRCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource
	r1 := resources[0]
	assert.Equal(t, "ecr", r1.Category)
	assert.Equal(t, "Repository", r1.SubCategory)
	assert.Equal(t, "my-app", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:ecr:us-east-1:123456789012:repository/my-app", r1.ARN)
	assert.Equal(t, "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app", helpers.GetMapValue(r1.RawData, "URI"))
	assert.Equal(t, "MUTABLE", helpers.GetMapValue(r1.RawData, "Mutability"))
	assert.Equal(t, "AES256", helpers.GetMapValue(r1.RawData, "Encryption"))
	assert.Equal(t, "5", helpers.GetMapValue(r1.RawData, "ImageCount"))
	assert.Equal(t, "2023-08-15T10:30:00Z", helpers.GetMapValue(r1.RawData, "CreatedAt"))

	// Check second resource
	r2 := resources[1]
	assert.Equal(t, "ecr", r2.Category)
	assert.Equal(t, "Repository", r2.SubCategory)
	assert.Equal(t, "web-service", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:ecr:us-east-1:123456789012:repository/web-service", r2.ARN)
	assert.Equal(t, "123456789012.dkr.ecr.us-east-1.amazonaws.com/web-service", helpers.GetMapValue(r2.RawData, "URI"))
	assert.Equal(t, "IMMUTABLE", helpers.GetMapValue(r2.RawData, "Mutability"))
	assert.Equal(t, "KMS", helpers.GetMapValue(r2.RawData, "Encryption"))
	assert.Equal(t, "12", helpers.GetMapValue(r2.RawData, "ImageCount"))
	assert.Equal(t, "2023-09-01T14:20:00Z", helpers.GetMapValue(r2.RawData, "CreatedAt"))
}
