package resources

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockCognitoCollector is a testable version of CognitoCollector that uses mock data
type MockCognitoCollector struct{}

func NewMockCognitoCollector() *MockCognitoCollector {
	return &MockCognitoCollector{}
}

func (c *MockCognitoCollector) Name() string {
	return "cognito"
}

func (c *MockCognitoCollector) ShouldSort() bool {
	return true
}

func (c *MockCognitoCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID
		{Header: "AllowUnauthenticated", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AllowUnauthenticated") }},
		{Header: "CreationDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationDate") }},
		{Header: "LastModifiedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastModifiedDate") }},
	}
}

func (c *MockCognitoCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock User Pool
	r1 := Resource{
		Category:    "cognito",
		SubCategory: "UserPool",
		Name:        "MyUserPool",
		Region:      region,
		ARN:         "us-east-1_ABC123DEF",
		RawData: helpers.NormalizeRawData(map[string]any{
			"LastModifiedDate": time.Date(2023, 8, 15, 10, 30, 0, 0, time.UTC),
			"CreationDate":     time.Date(2023, 8, 10, 9, 0, 0, 0, time.UTC),
		}),
	}
	resources = append(resources, r1)

	// Mock Identity Pool
	r2 := Resource{
		Category:    "cognito",
		SubCategory: "IdentityPool",
		Name:        "MyIdentityPool",
		Region:      region,
		ARN:         "us-east-1:12345678-1234-1234-1234-123456789012",
		RawData: helpers.NormalizeRawData(map[string]any{
			"AllowUnauthenticated": true,
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestCognitoCollector_Basic(t *testing.T) {
	collector := &CognitoCollector{}
	assert.Equal(t, "cognito", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestCognitoCollector_GetColumns(t *testing.T) {
	collector := &CognitoCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "AllowUnauthenticated", "CreationDate", "LastModifiedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "cognito",
		SubCategory:    "UserPool",
		SubSubCategory: "",
		Name:           "MyUserPool",
		Region:         "us-east-1",
		ARN:            "us-east-1_ABC123DEF",
		RawData: map[string]any{
			"AllowUnauthenticated": "false",
			"LastModifiedDate":     "2023-08-15T10:30:00Z",
			"CreationDate":         "2023-08-10T09:00:00Z",
		},
	}

	expectedValues := []string{
		"cognito", "UserPool", "", "MyUserPool", "us-east-1",
		"us-east-1_ABC123DEF", "false", "2023-08-10T09:00:00Z", "2023-08-15T10:30:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockCognitoCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockCognitoCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (User Pool)
	r1 := resources[0]
	assert.Equal(t, "cognito", r1.Category)
	assert.Equal(t, "UserPool", r1.SubCategory)
	assert.Equal(t, "MyUserPool", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "us-east-1_ABC123DEF", r1.ARN)
	assert.Equal(t, "2023-08-15T10:30:00Z", helpers.GetMapValue(r1.RawData, "LastModifiedDate"))
	assert.Equal(t, "2023-08-10T09:00:00Z", helpers.GetMapValue(r1.RawData, "CreationDate"))

	// Check second resource (Identity Pool)
	r2 := resources[1]
	assert.Equal(t, "cognito", r2.Category)
	assert.Equal(t, "IdentityPool", r2.SubCategory)
	assert.Equal(t, "MyIdentityPool", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "us-east-1:12345678-1234-1234-1234-123456789012", r2.ARN)
	assert.Equal(t, "true", helpers.GetMapValue(r2.RawData, "AllowUnauthenticated"))
}
