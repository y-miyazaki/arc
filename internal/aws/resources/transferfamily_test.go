package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockTransferFamilyCollector is a testable version of TransferFamilyCollector that uses mock data
type MockTransferFamilyCollector struct{}

func NewMockTransferFamilyCollector() *MockTransferFamilyCollector {
	return &MockTransferFamilyCollector{}
}

func (c *MockTransferFamilyCollector) Name() string {
	return "transferfamily"
}

func (c *MockTransferFamilyCollector) ShouldSort() bool {
	return true
}

func (c *MockTransferFamilyCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ServerID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ServerID
		{Header: "Protocol", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Protocol") }},
		{Header: "State", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "State") }},
	}
}

func (c *MockTransferFamilyCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Transfer Server
	r1 := Resource{
		Category:    "transferfamily",
		SubCategory: "Server",
		Name:        "s-1234567890abcdef0",
		Region:      region,
		ARN:         "s-1234567890abcdef0",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Protocol": "SFTP",
			"State":    "ONLINE",
		}),
	}
	resources = append(resources, r1)

	return resources, nil
}

func TestTransferFamilyCollector_Basic(t *testing.T) {
	collector := &TransferFamilyCollector{}
	assert.Equal(t, "transferfamily", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestTransferFamilyCollector_GetColumns(t *testing.T) {
	collector := &TransferFamilyCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ServerID", "Protocol", "State",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "Storage",
		SubCategory:    "Transfer Family",
		SubSubCategory: "Server",
		Name:           "s-1234567890abcdef0",
		Region:         "us-east-1",
		ARN:            "s-1234567890abcdef0",
		RawData: map[string]any{
			"Protocol": "SFTP",
			"State":    "ONLINE",
		},
	}

	expectedValues := []string{
		"Storage", "Transfer Family", "Server", "s-1234567890abcdef0", "us-east-1",
		"s-1234567890abcdef0", "SFTP", "ONLINE",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockTransferFamilyCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockTransferFamilyCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 1)

	// Check resource (Server)
	r1 := resources[0]
	assert.Equal(t, "transferfamily", r1.Category)
	assert.Equal(t, "Server", r1.SubCategory)
	assert.Equal(t, "s-1234567890abcdef0", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "s-1234567890abcdef0", r1.ARN)
	assert.Equal(t, "SFTP", helpers.GetMapValue(r1.RawData, "Protocol"))
	assert.Equal(t, "ONLINE", helpers.GetMapValue(r1.RawData, "State"))
}
