package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestEC2Collector_Name(t *testing.T) {
	collector := &EC2Collector{}
	assert.Equal(t, "ec2", collector.Name())
}

func TestEC2Collector_ShouldSort(t *testing.T) {
	collector := &EC2Collector{}
	assert.True(t, collector.ShouldSort())
}

func TestEC2Collector_GetColumns(t *testing.T) {
	collector := &EC2Collector{}
	columns := collector.GetColumns()

	assert.NotEmpty(t, columns)
	assert.Contains(t, columns[0].Header, "Category")
	assert.Contains(t, columns[1].Header, "SubCategory")
	assert.Contains(t, columns[2].Header, "SubSubCategory")
	assert.Contains(t, columns[3].Header, "Name")
	assert.Contains(t, columns[4].Header, "Region")
	assert.Contains(t, columns[5].Header, "InstanceID")
	assert.Contains(t, columns[6].Header, "InstanceType")

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "ec2",
		SubCategory:    "Instance",
		SubSubCategory: "t2.micro",
		Name:           "test-instance",
		Region:         "us-east-1",
		RawData: map[string]any{
			"InstanceID":    "i-1234567890abcdef0",
			"InstanceType":  "t2.micro",
			"State":         "running",
			"VPC":           "vpc-12345678",
			"Subnet":        "subnet-12345678",
			"SecurityGroup": "sg-12345678",
			"KeyName":       "my-key",
			"PublicIP":      "54.123.45.67",
			"PrivateIP":     "10.0.1.100",
		},
	}

	// Test each Value function
	assert.Equal(t, "ec2", columns[0].Value(sampleResource))
	assert.Equal(t, "Instance", columns[1].Value(sampleResource))
	assert.Equal(t, "t2.micro", columns[2].Value(sampleResource))
	assert.Equal(t, "test-instance", columns[3].Value(sampleResource))
	assert.Equal(t, "us-east-1", columns[4].Value(sampleResource))
	assert.Equal(t, "i-1234567890abcdef0", columns[5].Value(sampleResource))
	assert.Equal(t, "t2.micro", columns[6].Value(sampleResource))

	// Test Name field with empty name (should return "N/A")
	emptyNameResource := sampleResource
	emptyNameResource.Name = ""
	assert.Equal(t, "N/A", columns[3].Value(emptyNameResource))
}

// MockEC2Collector is a mock implementation of EC2Collector for testing
type MockEC2Collector struct{}

func (m *MockEC2Collector) Name() string {
	return "ec2"
}

func (m *MockEC2Collector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	return []Resource{
		{
			Category:    "ec2",
			SubCategory: "Instance",
			Name:        "test-instance",
			Region:      region,
			RawData: map[string]any{
				"InstanceID":    "i-1234567890abcdef0",
				"InstanceType":  "t2.micro",
				"ImageID":       "ami-12345678",
				"VPC":           "vpc-12345",
				"Subnet":        "subnet-12345",
				"SecurityGroup": "sg-1\nsg-2",
				"State":         "running",
			},
		},
	}, nil
}

func (m *MockEC2Collector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "InstanceID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "InstanceID") }},
		{Header: "InstanceType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "InstanceType") }},
	}
}

func (m *MockEC2Collector) ShouldSort() bool {
	return true
}

func TestMockEC2Collector_Collect(t *testing.T) {
	collector := &MockEC2Collector{}
	cfg := &aws.Config{}
	region := "us-east-1"

	resources, err := collector.Collect(context.Background(), cfg, region)

	assert.NoError(t, err)
	assert.NotEmpty(t, resources)
	assert.Equal(t, 1, len(resources))

	resource := resources[0]
	assert.Equal(t, "ec2", resource.Category)
	assert.Equal(t, "Instance", resource.SubCategory)
	assert.Equal(t, "test-instance", resource.Name)
	assert.Equal(t, region, resource.Region)
	assert.Equal(t, "i-1234567890abcdef0", helpers.GetMapValue(resource.RawData, "InstanceID"))
	assert.Equal(t, "t2.micro", helpers.GetMapValue(resource.RawData, "InstanceType"))
}
