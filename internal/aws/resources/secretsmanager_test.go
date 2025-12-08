package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockSecretsManagerCollector is a testable version of SecretsManagerCollector that uses mock data
type MockSecretsManagerCollector struct{}

func NewMockSecretsManagerCollector() *MockSecretsManagerCollector {
	return &MockSecretsManagerCollector{}
}

func (c *MockSecretsManagerCollector) Name() string {
	return "secretsmanager"
}

func (c *MockSecretsManagerCollector) ShouldSort() bool {
	return true
}

func (c *MockSecretsManagerCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "RotationEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RotationEnabled") }},
		{Header: "RotationLambdaARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RotationLambdaARN") }},
		{Header: "LastAccessedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastAccessedDate") }},
		{Header: "LastRotatedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastRotatedDate") }},
		{Header: "LastChangedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastChangedDate") }},
	}
}

func (c *MockSecretsManagerCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock secret 1
	r1 := Resource{
		Category:    "secretsmanager",
		SubCategory: "Secret",
		Name:        "prod/database/password",
		Region:      region,
		ARN:         "arn:aws:secretsmanager:us-east-1:123456789012:secret:prod/database/password-AbCdEf",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description":       "Production database password",
			"KmsKey":            "alias/aws/secretsmanager",
			"RotationEnabled":   "true",
			"RotationLambdaARN": "arn:aws:lambda:us-east-1:123456789012:function:rotate-secret",
			"LastAccessedDate":  "2023-09-20T08:30:00Z",
			"LastRotatedDate":   "2023-09-15T02:00:00Z",
			"LastChangedDate":   "2023-09-15T02:00:00Z",
		}),
	}
	resources = append(resources, r1)

	// Mock secret 2
	r2 := Resource{
		Category:    "secretsmanager",
		SubCategory: "Secret",
		Name:        "dev/api-key",
		Region:      region,
		ARN:         "arn:aws:secretsmanager:us-east-1:123456789012:secret:dev/api-key-GhIjKl",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description":       "Development API key",
			"KmsKey":            "alias/my-custom-key",
			"RotationEnabled":   "false",
			"RotationLambdaARN": "N/A",
			"LastAccessedDate":  "2023-09-25T14:15:00Z",
			"LastRotatedDate":   "N/A",
			"LastChangedDate":   "2023-08-30T10:00:00Z",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestSecretsManagerCollector_Basic(t *testing.T) {
	collector := &SecretsManagerCollector{}
	assert.Equal(t, "secretsmanager", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestSecretsManagerCollector_GetColumns(t *testing.T) {
	collector := &SecretsManagerCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Description", "KmsKey", "RotationEnabled", "RotationLambdaARN",
		"LastAccessedDate", "LastRotatedDate", "LastChangedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Security",
		SubCategory:    "Secrets Manager",
		SubSubCategory: "Secret",
		Name:           "my-secret",
		Region:         "us-east-1",
		ARN:            "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
		RawData: map[string]interface{}{
			"Description":       "Database password for production",
			"KmsKey":            "alias/aws/secretsmanager",
			"RotationEnabled":   "true",
			"RotationLambdaARN": "arn:aws:lambda:us-east-1:123456789012:function:rotate-secret",
			"LastAccessedDate":  "2023-06-15T10:30:00Z",
			"LastRotatedDate":   "2023-06-01T00:00:00Z",
			"LastChangedDate":   "2023-05-20T15:45:00Z",
		},
	}

	expectedValues := []string{
		"Security", "Secrets Manager", "Secret", "my-secret", "us-east-1",
		"arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
		"Database password for production", "alias/aws/secretsmanager", "true",
		"arn:aws:lambda:us-east-1:123456789012:function:rotate-secret",
		"2023-06-15T10:30:00Z", "2023-06-01T00:00:00Z", "2023-05-20T15:45:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}

func TestMockSecretsManagerCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockSecretsManagerCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource
	r1 := resources[0]
	assert.Equal(t, "secretsmanager", r1.Category)
	assert.Equal(t, "Secret", r1.SubCategory)
	assert.Equal(t, "prod/database/password", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:secretsmanager:us-east-1:123456789012:secret:prod/database/password-AbCdEf", r1.ARN)
	assert.Equal(t, "Production database password", helpers.GetMapValue(r1.RawData, "Description"))
	assert.Equal(t, "alias/aws/secretsmanager", helpers.GetMapValue(r1.RawData, "KmsKey"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "RotationEnabled"))
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:rotate-secret", helpers.GetMapValue(r1.RawData, "RotationLambdaARN"))
	assert.Equal(t, "2023-09-20T08:30:00Z", helpers.GetMapValue(r1.RawData, "LastAccessedDate"))
	assert.Equal(t, "2023-09-15T02:00:00Z", helpers.GetMapValue(r1.RawData, "LastRotatedDate"))
	assert.Equal(t, "2023-09-15T02:00:00Z", helpers.GetMapValue(r1.RawData, "LastChangedDate"))

	// Check second resource
	r2 := resources[1]
	assert.Equal(t, "secretsmanager", r2.Category)
	assert.Equal(t, "Secret", r2.SubCategory)
	assert.Equal(t, "dev/api-key", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:secretsmanager:us-east-1:123456789012:secret:dev/api-key-GhIjKl", r2.ARN)
	assert.Equal(t, "Development API key", helpers.GetMapValue(r2.RawData, "Description"))
	assert.Equal(t, "alias/my-custom-key", helpers.GetMapValue(r2.RawData, "KmsKey"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "RotationEnabled"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "RotationLambdaARN"))
	assert.Equal(t, "2023-09-25T14:15:00Z", helpers.GetMapValue(r2.RawData, "LastAccessedDate"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "LastRotatedDate"))
	assert.Equal(t, "2023-08-30T10:00:00Z", helpers.GetMapValue(r2.RawData, "LastChangedDate"))
}
