package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func (c *MockLambdaCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Runtime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Runtime") }},
		{Header: "Architecture", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Architecture") }},
		{Header: "MemorySize", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MemorySize") }},
		{Header: "Timeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Timeout") }},
		{Header: "EnvVars", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EnvVars") }},
		{Header: "LastModified", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastModified") }},
	}
}

func TestLambdaCollector_Basic(t *testing.T) {
	collector := &LambdaCollector{}
	assert.Equal(t, "lambda", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestLambdaCollector_GetColumns(t *testing.T) {
	collector := &LambdaCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "RoleARN", "Type", "Runtime", "Architecture",
		"MemorySize", "Timeout", "EnvVars", "LastModified",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "Compute",
		SubCategory:    "Lambda",
		SubSubCategory: "Function",
		Name:           "my-lambda-function",
		Region:         "us-east-1",
		ARN:            "arn:aws:lambda:us-east-1:123456789012:function:my-lambda-function",
		RawData: map[string]any{
			"RoleARN":      "arn:aws:iam::123456789012:role/lambda-role",
			"Type":         "Zip",
			"Runtime":      "python3.9",
			"Architecture": "x86_64",
			"MemorySize":   "128",
			"Timeout":      "30",
			"EnvVars":      "{\"KEY1\":\"value1\",\"KEY2\":\"value2\"}",
			"LastModified": "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Compute", "Lambda", "Function", "my-lambda-function", "us-east-1",
		"arn:aws:lambda:us-east-1:123456789012:function:my-lambda-function",
		"arn:aws:iam::123456789012:role/lambda-role", "Zip", "python3.9", "x86_64",
		"128", "30", "{\"KEY1\":\"value1\",\"KEY2\":\"value2\"}", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

// MockLambdaCollector is a testable version of LambdaCollector that uses mock data
type MockLambdaCollector struct{}

func NewMockLambdaCollector() *MockLambdaCollector {
	return &MockLambdaCollector{}
}

func (c *MockLambdaCollector) Name() string {
	return "lambda"
}

func (c *MockLambdaCollector) ShouldSort() bool {
	return true
}

func (c *MockLambdaCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock function 1
	r1 := Resource{
		Category:    "lambda",
		SubCategory: "Function",
		Name:        "my-api-handler",
		Region:      region,
		ARN:         "arn:aws:lambda:us-east-1:123456789012:function:my-api-handler",
		RawData: helpers.NormalizeRawData(map[string]any{
			"RoleARN":      "arn:aws:iam::123456789012:role/lambda-execution-role",
			"Type":         "Zip",
			"Runtime":      "nodejs18.x",
			"Architecture": "x86_64",
			"MemorySize":   "256",
			"Timeout":      "30",
			"EnvVars":      "3",
			"LastModified": "2023-09-15T08:30:00Z",
		}),
	}
	resources = append(resources, r1)

	// Mock function 2
	r2 := Resource{
		Category:    "lambda",
		SubCategory: "Function",
		Name:        "data-processor",
		Region:      region,
		ARN:         "arn:aws:lambda:us-east-1:123456789012:function:data-processor",
		RawData: helpers.NormalizeRawData(map[string]any{
			"RoleARN":      "arn:aws:iam::123456789012:role/lambda-processor-role",
			"Type":         "Zip",
			"Runtime":      "python3.9",
			"Architecture": "arm64",
			"MemorySize":   "512",
			"Timeout":      "300",
			"EnvVars":      "5",
			"LastModified": "2023-08-20T14:15:00Z",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestMockLambdaCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockLambdaCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource
	r1 := resources[0]
	assert.Equal(t, "lambda", r1.Category)
	assert.Equal(t, "Function", r1.SubCategory)
	assert.Equal(t, "my-api-handler", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:my-api-handler", r1.ARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/lambda-execution-role", helpers.GetMapValue(r1.RawData, "RoleARN"))
	assert.Equal(t, "Zip", helpers.GetMapValue(r1.RawData, "Type"))
	assert.Equal(t, "nodejs18.x", helpers.GetMapValue(r1.RawData, "Runtime"))
	assert.Equal(t, "x86_64", helpers.GetMapValue(r1.RawData, "Architecture"))
	assert.Equal(t, "256", helpers.GetMapValue(r1.RawData, "MemorySize"))
	assert.Equal(t, "30", helpers.GetMapValue(r1.RawData, "Timeout"))
	assert.Equal(t, "3", helpers.GetMapValue(r1.RawData, "EnvVars"))
	assert.Equal(t, "2023-09-15T08:30:00Z", helpers.GetMapValue(r1.RawData, "LastModified"))

	// Check second resource
	r2 := resources[1]
	assert.Equal(t, "lambda", r2.Category)
	assert.Equal(t, "Function", r2.SubCategory)
	assert.Equal(t, "data-processor", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:data-processor", r2.ARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/lambda-processor-role", helpers.GetMapValue(r2.RawData, "RoleARN"))
	assert.Equal(t, "Zip", helpers.GetMapValue(r2.RawData, "Type"))
	assert.Equal(t, "python3.9", helpers.GetMapValue(r2.RawData, "Runtime"))
	assert.Equal(t, "arm64", helpers.GetMapValue(r2.RawData, "Architecture"))
	assert.Equal(t, "512", helpers.GetMapValue(r2.RawData, "MemorySize"))
	assert.Equal(t, "300", helpers.GetMapValue(r2.RawData, "Timeout"))
	assert.Equal(t, "5", helpers.GetMapValue(r2.RawData, "EnvVars"))
	assert.Equal(t, "2023-08-20T14:15:00Z", helpers.GetMapValue(r2.RawData, "LastModified"))
}
