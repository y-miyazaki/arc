package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestVPCCollector_Name(t *testing.T) {
	collector := &VPCCollector{}
	assert.Equal(t, "vpc", collector.Name())
}

func TestVPCCollector_ShouldSort(t *testing.T) {
	collector := &VPCCollector{}
	assert.False(t, collector.ShouldSort())
}

func TestVPCCollector_GetColumns(t *testing.T) {
	collector := &VPCCollector{}
	columns := collector.GetColumns()

	assert.NotEmpty(t, columns)
	assert.Contains(t, columns[0].Header, "Category")
	assert.Contains(t, columns[1].Header, "SubCategory")
	assert.Contains(t, columns[2].Header, "SubSubCategory")
	assert.Contains(t, columns[3].Header, "Name")
	assert.Contains(t, columns[4].Header, "Region")
	assert.Contains(t, columns[5].Header, "ID")

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "Networking",
		SubCategory:    "VPC",
		SubSubCategory: "VPC",
		Name:           "my-vpc",
		Region:         "us-east-1",
		RawData: map[string]any{
			"ID": "vpc-12345678",
		},
	}

	// Test basic columns
	assert.Equal(t, "Networking", columns[0].Value(sampleResource))
	assert.Equal(t, "VPC", columns[1].Value(sampleResource))
	assert.Equal(t, "VPC", columns[2].Value(sampleResource))
	assert.Equal(t, "my-vpc", columns[3].Value(sampleResource))
	assert.Equal(t, "us-east-1", columns[4].Value(sampleResource))
	assert.Equal(t, "vpc-12345678", columns[5].Value(sampleResource))

	// Test Name field with empty value
	emptyNameResource := sampleResource
	emptyNameResource.Name = ""
	assert.Equal(t, "N/A", columns[3].Value(emptyNameResource))
}

// MockVPCCollector is a mock implementation of VPCCollector for testing
type MockVPCCollector struct{}

func (m *MockVPCCollector) Name() string {
	return "vpc"
}

func (m *MockVPCCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	return []Resource{
		{
			Category:    "vpc",
			SubCategory: "VPC",
			Name:        "test-vpc",
			Region:      region,
			RawData: map[string]any{
				"ID":        "vpc-12345678",
				"CIDR":      "10.0.0.0/16",
				"State":     "available",
				"IsDefault": "false",
			},
		},
		{
			Category:    "vpc",
			SubCategory: "Subnet",
			Name:        "test-subnet",
			Region:      region,
			RawData: map[string]any{
				"ID":               "subnet-12345678",
				"VPCID":            "vpc-12345678",
				"CIDR":             "10.0.1.0/24",
				"AvailabilityZone": "us-east-1a",
				"State":            "available",
			},
		},
	}, nil
}

func (m *MockVPCCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ID") }},
	}
}

func (m *MockVPCCollector) ShouldSort() bool {
	return false
}

func TestMockVPCCollector_Collect(t *testing.T) {
	collector := &MockVPCCollector{}
	cfg := &aws.Config{}
	region := "us-east-1"

	resources, err := collector.Collect(context.Background(), cfg, region)

	assert.NoError(t, err)
	assert.NotEmpty(t, resources)
	assert.Equal(t, 2, len(resources))

	// Check VPC resource
	vpcResource := resources[0]
	assert.Equal(t, "vpc", vpcResource.Category)
	assert.Equal(t, "VPC", vpcResource.SubCategory)
	assert.Equal(t, "test-vpc", vpcResource.Name)
	assert.Equal(t, region, vpcResource.Region)

	// Check subnet resource
	subnetResource := resources[1]
	assert.Equal(t, "vpc", subnetResource.Category)
	assert.Equal(t, "Subnet", subnetResource.SubCategory)
	assert.Equal(t, "test-subnet", subnetResource.Name)
	assert.Equal(t, region, subnetResource.Region)
}
