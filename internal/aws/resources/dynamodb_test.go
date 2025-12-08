package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockDynamoDBCollector is a testable version of DynamoDBCollector that uses mock data
type MockDynamoDBCollector struct{}

func NewMockDynamoDBCollector() *MockDynamoDBCollector {
	return &MockDynamoDBCollector{}
}

func (c *MockDynamoDBCollector) Name() string {
	return "dynamodb"
}

func (c *MockDynamoDBCollector) ShouldSort() bool {
	return true
}

func (c *MockDynamoDBCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "AttributeDefinitions", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttributeDefinitions") }},
		{Header: "BillingMode", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "BillingMode") }},
		{Header: "StreamEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "StreamEnabled") }},
		{Header: "GlobalTable", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "GlobalTable") }},
		{Header: "PointInTimeRecovery", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PointInTimeRecovery") }},
		{Header: "RecoveryPeriodInDays", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RecoveryPeriodInDays") }},
		{Header: "EarliestRestorableDateTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EarliestRestorableDateTime") }},
		{Header: "LatestRestorableDateTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LatestRestorableDateTime") }},
		{Header: "DeletionProtection", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DeletionProtection") }},
		{Header: "TTLAttribute", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TTLAttribute") }},
		{Header: "SSE", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SSE") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "ItemCount", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ItemCount") }},
		{Header: "TableSize(Bytes)", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "TableSize(Bytes)") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
		{Header: "CreationDateTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreationDateTime") }},
	}
}

func (c *MockDynamoDBCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock table 1
	r1 := Resource{
		Category:    "dynamodb",
		SubCategory: "Table",
		Name:        "users-table",
		Region:      region,
		ARN:         "arn:aws:dynamodb:us-east-1:123456789012:table/users-table",
		RawData: helpers.NormalizeRawData(map[string]any{
			"AttributeDefinitions":       "id:S",
			"BillingMode":                "PAY_PER_REQUEST",
			"StreamEnabled":              "false",
			"GlobalTable":                "false",
			"PointInTimeRecovery":        "ENABLED",
			"RecoveryPeriodInDays":       "35",
			"EarliestRestorableDateTime": "2023-08-01T00:00:00Z",
			"LatestRestorableDateTime":   "2023-09-25T12:00:00Z",
			"DeletionProtection":         "ENABLED",
			"TTLAttribute":               "N/A",
			"SSE":                        "ENABLED",
			"KmsKey":                     "alias/aws/dynamodb",
			"ItemCount":                  "1500",
			"TableSize(Bytes)":           "1048576",
			"Status":                     "ACTIVE",
			"CreationDateTime":           "2023-01-15T10:30:00Z",
		}),
	}
	resources = append(resources, r1)

	// Mock table 2
	r2 := Resource{
		Category:    "dynamodb",
		SubCategory: "Table",
		Name:        "orders-table",
		Region:      region,
		ARN:         "arn:aws:dynamodb:us-east-1:123456789012:table/orders-table",
		RawData: helpers.NormalizeRawData(map[string]any{
			"AttributeDefinitions":       "orderId:S,userId:S",
			"BillingMode":                "PROVISIONED",
			"StreamEnabled":              "true",
			"GlobalTable":                "true",
			"PointInTimeRecovery":        "DISABLED",
			"RecoveryPeriodInDays":       "N/A",
			"EarliestRestorableDateTime": "N/A",
			"LatestRestorableDateTime":   "N/A",
			"DeletionProtection":         "DISABLED",
			"TTLAttribute":               "expiresAt",
			"SSE":                        "ENABLED",
			"KmsKey":                     "alias/my-key",
			"ItemCount":                  "5000",
			"TableSize(Bytes)":           "2097152",
			"Status":                     "ACTIVE",
			"CreationDateTime":           "2023-02-20T14:45:00Z",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestDynamoDBCollector_Basic(t *testing.T) {
	collector := &DynamoDBCollector{}
	assert.Equal(t, "dynamodb", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestDynamoDBCollector_GetColumns(t *testing.T) {
	collector := &DynamoDBCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "AttributeDefinitions", "BillingMode", "StreamEnabled",
		"GlobalTable", "PointInTimeRecovery", "RecoveryPeriodInDays",
		"EarliestRestorableDateTime", "LatestRestorableDateTime",
		"DeletionProtection", "TTLAttribute", "SSE", "KmsKey",
		"ItemCount", "TableSize(Bytes)", "Status", "CreationDateTime",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "dynamodb",
		SubCategory:    "Table",
		SubSubCategory: "",
		Name:           "users-table",
		Region:         "us-east-1",
		ARN:            "arn:aws:dynamodb:us-east-1:123456789012:table/users-table",
		RawData: map[string]any{
			"AttributeDefinitions":       "userId:S,email:S",
			"BillingMode":                "PAY_PER_REQUEST",
			"StreamEnabled":              "false",
			"GlobalTable":                "false",
			"PointInTimeRecovery":        "ENABLED",
			"RecoveryPeriodInDays":       "35",
			"EarliestRestorableDateTime": "2023-01-01T00:00:00Z",
			"LatestRestorableDateTime":   "2023-12-31T23:59:59Z",
			"DeletionProtection":         "ENABLED",
			"TTLAttribute":               "ttl",
			"SSE":                        "ENABLED",
			"KmsKey":                     "alias/aws/dynamodb",
			"ItemCount":                  "10000",
			"TableSize":                  "1048576",
			"Status":                     "ACTIVE",
			"CreationDateTime":           "2023-01-15T10:30:00Z",
		},
	}

	expectedValues := []string{
		"dynamodb", "Table", "", "users-table", "us-east-1",
		"arn:aws:dynamodb:us-east-1:123456789012:table/users-table",
		"userId:S,email:S", "PAY_PER_REQUEST", "false", "false", "ENABLED", "35",
		"2023-01-01T00:00:00Z", "2023-12-31T23:59:59Z", "ENABLED", "ttl",
		"ENABLED", "alias/aws/dynamodb", "10000", "1048576", "ACTIVE", "2023-01-15T10:30:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockDynamoDBCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockDynamoDBCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource
	r1 := resources[0]
	assert.Equal(t, "dynamodb", r1.Category)
	assert.Equal(t, "Table", r1.SubCategory)
	assert.Equal(t, "users-table", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:dynamodb:us-east-1:123456789012:table/users-table", r1.ARN)
	assert.Equal(t, "id:S", helpers.GetMapValue(r1.RawData, "AttributeDefinitions"))
	assert.Equal(t, "PAY_PER_REQUEST", helpers.GetMapValue(r1.RawData, "BillingMode"))
	assert.Equal(t, "false", helpers.GetMapValue(r1.RawData, "StreamEnabled"))
	assert.Equal(t, "false", helpers.GetMapValue(r1.RawData, "GlobalTable"))
	assert.Equal(t, "ENABLED", helpers.GetMapValue(r1.RawData, "PointInTimeRecovery"))
	assert.Equal(t, "35", helpers.GetMapValue(r1.RawData, "RecoveryPeriodInDays"))
	assert.Equal(t, "2023-08-01T00:00:00Z", helpers.GetMapValue(r1.RawData, "EarliestRestorableDateTime"))
	assert.Equal(t, "2023-09-25T12:00:00Z", helpers.GetMapValue(r1.RawData, "LatestRestorableDateTime"))
	assert.Equal(t, "ENABLED", helpers.GetMapValue(r1.RawData, "DeletionProtection"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r1.RawData, "TTLAttribute"))
	assert.Equal(t, "ENABLED", helpers.GetMapValue(r1.RawData, "SSE"))
	assert.Equal(t, "alias/aws/dynamodb", helpers.GetMapValue(r1.RawData, "KmsKey"))
	assert.Equal(t, "1500", helpers.GetMapValue(r1.RawData, "ItemCount"))
	assert.Equal(t, "1048576", helpers.GetMapValue(r1.RawData, "TableSize(Bytes)"))
	assert.Equal(t, "ACTIVE", helpers.GetMapValue(r1.RawData, "Status"))

	// Check second resource
	r2 := resources[1]
	assert.Equal(t, "dynamodb", r2.Category)
	assert.Equal(t, "Table", r2.SubCategory)
	assert.Equal(t, "orders-table", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:dynamodb:us-east-1:123456789012:table/orders-table", r2.ARN)
	assert.Equal(t, "orderId:S,userId:S", helpers.GetMapValue(r2.RawData, "AttributeDefinitions"))
	assert.Equal(t, "PROVISIONED", helpers.GetMapValue(r2.RawData, "BillingMode"))
	assert.Equal(t, "true", helpers.GetMapValue(r2.RawData, "StreamEnabled"))
	assert.Equal(t, "true", helpers.GetMapValue(r2.RawData, "GlobalTable"))
	assert.Equal(t, "DISABLED", helpers.GetMapValue(r2.RawData, "PointInTimeRecovery"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "RecoveryPeriodInDays"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "EarliestRestorableDateTime"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "LatestRestorableDateTime"))
	assert.Equal(t, "DISABLED", helpers.GetMapValue(r2.RawData, "DeletionProtection"))
	assert.Equal(t, "expiresAt", helpers.GetMapValue(r2.RawData, "TTLAttribute"))
	assert.Equal(t, "ENABLED", helpers.GetMapValue(r2.RawData, "SSE"))
	assert.Equal(t, "alias/my-key", helpers.GetMapValue(r2.RawData, "KmsKey"))
	assert.Equal(t, "5000", helpers.GetMapValue(r2.RawData, "ItemCount"))
	assert.Equal(t, "2097152", helpers.GetMapValue(r2.RawData, "TableSize(Bytes)"))
	assert.Equal(t, "ACTIVE", helpers.GetMapValue(r2.RawData, "Status"))
}
