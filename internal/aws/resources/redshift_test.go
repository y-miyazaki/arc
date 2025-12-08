package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockRedshiftCollector is a testable version of RedshiftCollector that uses mock data
type MockRedshiftCollector struct{}

func NewMockRedshiftCollector() *MockRedshiftCollector {
	return &MockRedshiftCollector{}
}

func (c *MockRedshiftCollector) Name() string {
	return "redshift"
}

func (c *MockRedshiftCollector) ShouldSort() bool {
	return true
}

func (c *MockRedshiftCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "NodeType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NodeType") }},
		{Header: "NumberOfNodes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NumberOfNodes") }},
		{Header: "DBName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DBName") }},
		{Header: "Endpoint", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Endpoint") }},
		{Header: "Port", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Port") }},
		{Header: "MasterUsername", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MasterUsername") }},
		{Header: "VPCName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "VPCName") }},
		{Header: "ClusterSubnetGroupName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ClusterSubnetGroupName") }},
		{Header: "SecurityGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityGroup") }},
		{Header: "Encrypted", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Encrypted") }},
		{Header: "KmsKey", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "KmsKey") }},
		{Header: "PubliclyAccessible", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PubliclyAccessible") }},
		{Header: "ClusterStatus", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ClusterStatus") }},
	}
}

func (c *MockRedshiftCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock cluster 1
	r1 := Resource{
		Category:    "redshift",
		SubCategory: "Cluster",
		Name:        "my-data-warehouse",
		Region:      region,
		ARN:         "arn:aws:redshift:us-east-1:123456789012:cluster:my-data-warehouse",
		RawData: helpers.NormalizeRawData(map[string]any{
			"RoleARN":                "arn:aws:iam::123456789012:role/RedshiftRole",
			"NodeType":               "dc2.large",
			"NumberOfNodes":          "3",
			"DBName":                 "analytics",
			"Endpoint":               "my-data-warehouse.cluster-abc123xyz.us-east-1.redshift.amazonaws.com",
			"Port":                   "5439",
			"MasterUsername":         "admin",
			"VPCName":                "vpc-12345",
			"ClusterSubnetGroupName": "default",
			"SecurityGroup":          "sg-12345,sg-67890",
			"Encrypted":              "true",
			"KmsKey":                 "alias/aws/redshift",
			"PubliclyAccessible":     "false",
			"ClusterStatus":          "available",
		}),
	}
	resources = append(resources, r1)

	// Mock cluster 2
	r2 := Resource{
		Category:    "redshift",
		SubCategory: "Cluster",
		Name:        "test-cluster",
		Region:      region,
		ARN:         "arn:aws:redshift:us-east-1:123456789012:cluster:test-cluster",
		RawData: helpers.NormalizeRawData(map[string]any{
			"RoleARN":                "N/A",
			"NodeType":               "ra3.xlplus",
			"NumberOfNodes":          "1",
			"DBName":                 "testdb",
			"Endpoint":               "test-cluster.cluster-def456uvw.us-east-1.redshift.amazonaws.com",
			"Port":                   "5439",
			"MasterUsername":         "testuser",
			"VPCName":                "vpc-67890",
			"ClusterSubnetGroupName": "test-subnet-group",
			"SecurityGroup":          "sg-54321",
			"Encrypted":              "false",
			"KmsKey":                 "N/A",
			"PubliclyAccessible":     "true",
			"ClusterStatus":          "available",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestRedshiftCollector_Basic(t *testing.T) {
	collector := &RedshiftCollector{}
	assert.Equal(t, "redshift", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestRedshiftCollector_GetColumns(t *testing.T) {
	collector := &RedshiftCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "RoleARN",
		"NodeType", "NumberOfNodes", "DBName", "Endpoint", "Port", "MasterUsername",
		"VPCName", "ClusterSubnetGroupName", "SecurityGroup", "Encrypted", "KmsKey", "PubliclyAccessible", "ClusterStatus",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "redshift",
		SubCategory:    "Cluster",
		SubSubCategory: "",
		Name:           "analytics-cluster",
		Region:         "us-east-1",
		ARN:            "arn:aws:redshift:us-east-1:123456789012:cluster:analytics-cluster",
		RawData: map[string]any{
			"NodeType":               "ra3.xlplus",
			"NumberOfNodes":          "3",
			"DBName":                 "analytics",
			"Endpoint":               "analytics-cluster.cluster-abc123def.us-east-1.redshift.amazonaws.com",
			"Port":                   "5439",
			"MasterUsername":         "admin",
			"VPCName":                "vpc-12345",
			"ClusterSubnetGroupName": "analytics-subnet-group",
			"SecurityGroup":          "sg-67890",
			"Encrypted":              "true",
			"KmsKey":                 "alias/redshift-key",
			"PubliclyAccessible":     "false",
			"ClusterStatus":          "available",
		},
	}

	expectedValues := []string{
		"redshift", "Cluster", "", "analytics-cluster", "us-east-1",
		"arn:aws:redshift:us-east-1:123456789012:cluster:analytics-cluster",
		"ra3.xlplus", "3", "analytics",
		"analytics-cluster.cluster-abc123def.us-east-1.redshift.amazonaws.com",
		"5439", "admin", "vpc-12345", "analytics-subnet-group", "sg-67890",
		"true", "alias/redshift-key", "false", "available",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockRedshiftCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockRedshiftCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource
	r1 := resources[0]
	assert.Equal(t, "redshift", r1.Category)
	assert.Equal(t, "Cluster", r1.SubCategory)
	assert.Equal(t, "my-data-warehouse", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:redshift:us-east-1:123456789012:cluster:my-data-warehouse", r1.ARN)
	assert.Equal(t, "arn:aws:iam::123456789012:role/RedshiftRole", helpers.GetMapValue(r1.RawData, "RoleARN"))
	assert.Equal(t, "dc2.large", helpers.GetMapValue(r1.RawData, "NodeType"))
	assert.Equal(t, "3", helpers.GetMapValue(r1.RawData, "NumberOfNodes"))
	assert.Equal(t, "analytics", helpers.GetMapValue(r1.RawData, "DBName"))
	assert.Equal(t, "my-data-warehouse.cluster-abc123xyz.us-east-1.redshift.amazonaws.com", helpers.GetMapValue(r1.RawData, "Endpoint"))
	assert.Equal(t, "5439", helpers.GetMapValue(r1.RawData, "Port"))
	assert.Equal(t, "admin", helpers.GetMapValue(r1.RawData, "MasterUsername"))
	assert.Equal(t, "vpc-12345", helpers.GetMapValue(r1.RawData, "VPCName"))
	assert.Equal(t, "default", helpers.GetMapValue(r1.RawData, "ClusterSubnetGroupName"))
	assert.Equal(t, "sg-12345,sg-67890", helpers.GetMapValue(r1.RawData, "SecurityGroup"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "Encrypted"))
	assert.Equal(t, "alias/aws/redshift", helpers.GetMapValue(r1.RawData, "KmsKey"))
	assert.Equal(t, "false", helpers.GetMapValue(r1.RawData, "PubliclyAccessible"))
	assert.Equal(t, "available", helpers.GetMapValue(r1.RawData, "ClusterStatus"))

	// Check second resource
	r2 := resources[1]
	assert.Equal(t, "redshift", r2.Category)
	assert.Equal(t, "Cluster", r2.SubCategory)
	assert.Equal(t, "test-cluster", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:redshift:us-east-1:123456789012:cluster:test-cluster", r2.ARN)
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "RoleARN"))
	assert.Equal(t, "ra3.xlplus", helpers.GetMapValue(r2.RawData, "NodeType"))
	assert.Equal(t, "1", helpers.GetMapValue(r2.RawData, "NumberOfNodes"))
	assert.Equal(t, "testdb", helpers.GetMapValue(r2.RawData, "DBName"))
	assert.Equal(t, "test-cluster.cluster-def456uvw.us-east-1.redshift.amazonaws.com", helpers.GetMapValue(r2.RawData, "Endpoint"))
	assert.Equal(t, "5439", helpers.GetMapValue(r2.RawData, "Port"))
	assert.Equal(t, "testuser", helpers.GetMapValue(r2.RawData, "MasterUsername"))
	assert.Equal(t, "vpc-67890", helpers.GetMapValue(r2.RawData, "VPCName"))
	assert.Equal(t, "test-subnet-group", helpers.GetMapValue(r2.RawData, "ClusterSubnetGroupName"))
	assert.Equal(t, "sg-54321", helpers.GetMapValue(r2.RawData, "SecurityGroup"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "Encrypted"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "KmsKey"))
	assert.Equal(t, "true", helpers.GetMapValue(r2.RawData, "PubliclyAccessible"))
	assert.Equal(t, "available", helpers.GetMapValue(r2.RawData, "ClusterStatus"))
}
