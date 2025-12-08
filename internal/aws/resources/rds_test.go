package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func (c *MockRDSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "Engine", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Engine") }},
		{Header: "Version", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Version") }},
	}
}

func TestRDSCollector_Basic(t *testing.T) {
	collector := &RDSCollector{}
	assert.Equal(t, "rds", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestRDSCollector_GetColumns(t *testing.T) {
	collector := &RDSCollector{}
	columns := collector.GetColumns()

	// RDS has many columns, just check the first few
	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "Type", "Engine", "Version",
	}

	assert.True(t, len(columns) >= len(expectedHeaders))
	for i, expected := range expectedHeaders {
		assert.Equal(t, expected, columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "Database",
		SubCategory:    "RDS",
		SubSubCategory: "DBInstance",
		Name:           "my-db-instance",
		Region:         "us-east-1",
		RawData: map[string]any{
			"ID":      "db-1234567890abcdef0",
			"Type":    "db.t3.micro",
			"Engine":  "mysql",
			"Version": "8.0.28",
		},
	}

	expectedValues := []string{
		"Database", "RDS", "DBInstance", "my-db-instance", "us-east-1",
		"db-1234567890abcdef0", "db.t3.micro", "mysql", "8.0.28",
	}

	for i, expected := range expectedValues {
		assert.Equal(t, expected, columns[i].Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

// MockRDSCollector is a testable version of RDSCollector that uses mock data
type MockRDSCollector struct{}

func NewMockRDSCollector() *MockRDSCollector {
	return &MockRDSCollector{}
}

func (c *MockRDSCollector) Name() string {
	return "rds"
}

func (c *MockRDSCollector) ShouldSort() bool {
	return false
}

func (c *MockRDSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock DB instance
	r1 := Resource{
		Category:    "rds",
		SubCategory: "DBInstance",
		Name:        "my-database",
		Region:      region,
		ARN:         "arn:aws:rds:us-east-1:123456789012:db:my-database",
		RawData: helpers.NormalizeRawData(map[string]any{
			"ID":      "my-database",
			"Type":    "db.t3.micro",
			"Engine":  "mysql",
			"Version": "8.0.32",
		}),
	}
	resources = append(resources, r1)

	// Mock DB cluster
	r2 := Resource{
		Category:    "rds",
		SubCategory: "DBCluster",
		Name:        "my-cluster",
		Region:      region,
		ARN:         "arn:aws:rds:us-east-1:123456789012:cluster:my-cluster",
		RawData: helpers.NormalizeRawData(map[string]any{
			"ID":      "my-cluster",
			"Type":    "db.r6g.large",
			"Engine":  "aurora-mysql",
			"Version": "8.0.mysql_aurora.3.02.0",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestMockRDSCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockRDSCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (DB Instance)
	r1 := resources[0]
	assert.Equal(t, "rds", r1.Category)
	assert.Equal(t, "DBInstance", r1.SubCategory)
	assert.Equal(t, "my-database", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:rds:us-east-1:123456789012:db:my-database", r1.ARN)
	assert.Equal(t, "my-database", helpers.GetMapValue(r1.RawData, "ID"))
	assert.Equal(t, "db.t3.micro", helpers.GetMapValue(r1.RawData, "Type"))
	assert.Equal(t, "mysql", helpers.GetMapValue(r1.RawData, "Engine"))
	assert.Equal(t, "8.0.32", helpers.GetMapValue(r1.RawData, "Version"))

	// Check second resource (DB Cluster)
	r2 := resources[1]
	assert.Equal(t, "rds", r2.Category)
	assert.Equal(t, "DBCluster", r2.SubCategory)
	assert.Equal(t, "my-cluster", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:rds:us-east-1:123456789012:cluster:my-cluster", r2.ARN)
	assert.Equal(t, "my-cluster", helpers.GetMapValue(r2.RawData, "ID"))
	assert.Equal(t, "db.r6g.large", helpers.GetMapValue(r2.RawData, "Type"))
	assert.Equal(t, "aurora-mysql", helpers.GetMapValue(r2.RawData, "Engine"))
	assert.Equal(t, "8.0.mysql_aurora.3.02.0", helpers.GetMapValue(r2.RawData, "Version"))
}
