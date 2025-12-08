package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockKMSCollector is a testable version of KMSCollector that uses mock data
type MockKMSCollector struct{}

func NewMockKMSCollector() *MockKMSCollector {
	return &MockKMSCollector{}
}

func (c *MockKMSCollector) Name() string {
	return "kms"
}

func (c *MockKMSCollector) ShouldSort() bool {
	return true
}

func (c *MockKMSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "KeyUsage", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KeyUsage") }},
		{Header: "KeyManager", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KeyManager") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

func (c *MockKMSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock KMS Key
	r1 := Resource{
		Category:    "kms",
		SubCategory: "Key",
		Name:        "alias/my-encryption-key",
		Region:      region,
		ARN:         "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description": "Key for encrypting sensitive data",
			"KeyUsage":    "ENCRYPT_DECRYPT",
			"KeyManager":  "CUSTOMER",
			"State":       "Enabled",
		}),
	}
	resources = append(resources, r1)

	return resources, nil
}

func TestKMSCollector_Basic(t *testing.T) {
	collector := &KMSCollector{}
	assert.Equal(t, "kms", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestKMSCollector_GetColumns(t *testing.T) {
	collector := &KMSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Description", "KeyUsage", "KeyManager", "State",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "kms",
		SubCategory:    "Key",
		SubSubCategory: "",
		Name:           "alias/my-app-key",
		Region:         "us-east-1",
		ARN:            "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
		RawData: map[string]any{
			"Description": "Encryption key for application data",
			"KeyUsage":    "ENCRYPT_DECRYPT",
			"KeyManager":  "CUSTOMER",
			"State":       "Enabled",
		},
	}

	expectedValues := []string{
		"kms", "Key", "", "alias/my-app-key", "us-east-1",
		"arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
		"Encryption key for application data", "ENCRYPT_DECRYPT", "CUSTOMER", "Enabled",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockKMSCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockKMSCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 1)

	// Check resource (Key)
	r1 := resources[0]
	assert.Equal(t, "kms", r1.Category)
	assert.Equal(t, "Key", r1.SubCategory)
	assert.Equal(t, "alias/my-encryption-key", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", r1.ARN)
	assert.Equal(t, "Key for encrypting sensitive data", helpers.GetMapValue(r1.RawData, "Description"))
	assert.Equal(t, "ENCRYPT_DECRYPT", helpers.GetMapValue(r1.RawData, "KeyUsage"))
	assert.Equal(t, "CUSTOMER", helpers.GetMapValue(r1.RawData, "KeyManager"))
	assert.Equal(t, "Enabled", helpers.GetMapValue(r1.RawData, "State"))
}
