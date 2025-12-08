package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestEFSCollector_Name(t *testing.T) {
	collector := &EFSCollector{}
	assert.Equal(t, "efs", collector.Name())
}

func TestEFSCollector_ShouldSort(t *testing.T) {
	collector := &EFSCollector{}
	assert.True(t, collector.ShouldSort())
}

func TestEFSCollector_GetColumns(t *testing.T) {
	collector := &EFSCollector{}
	columns := collector.GetColumns()

	assert.NotEmpty(t, columns)
	assert.Contains(t, columns[0].Header, "Category")
	assert.Contains(t, columns[1].Header, "SubCategory")
	assert.Contains(t, columns[2].Header, "SubSubCategory")
	assert.Contains(t, columns[3].Header, "Name")
	assert.Contains(t, columns[4].Header, "Region")
	assert.Contains(t, columns[5].Header, "ID")
	assert.Contains(t, columns[6].Header, "Type")

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "efs",
		SubCategory:    "FileSystem",
		SubSubCategory: "MountTarget",
		Name:           "my-filesystem",
		Region:         "us-east-1",
		ARN:            "fs-12345678",
		RawData: map[string]any{
			"Type":          "MountTarget",
			"Performance":   "generalPurpose",
			"Throughput":    "bursting",
			"Encrypted":     "true",
			"Size":          "5368709120",
			"Subnet":        "subnet-12345678",
			"IPAddress":     "10.0.1.100",
			"SecurityGroup": []string{"sg-12345678"},
			"Path":          "/mnt/efs",
			"UID":           "1000",
		},
	}

	// Test each Value function
	assert.Equal(t, "efs", columns[0].Value(sampleResource))
	assert.Equal(t, "FileSystem", columns[1].Value(sampleResource))
	assert.Equal(t, "MountTarget", columns[2].Value(sampleResource))
	assert.Equal(t, "my-filesystem", columns[3].Value(sampleResource))
	assert.Equal(t, "us-east-1", columns[4].Value(sampleResource))
	assert.Equal(t, "fs-12345678", columns[5].Value(sampleResource)) // ID uses ARN
	assert.Equal(t, "MountTarget", columns[6].Value(sampleResource))
	assert.Equal(t, "generalPurpose", columns[7].Value(sampleResource))
	assert.Equal(t, "bursting", columns[8].Value(sampleResource))
	assert.Equal(t, "true", columns[9].Value(sampleResource))
	// columns[10] is KmsKey
	assert.Equal(t, "5368709120", columns[11].Value(sampleResource))
	assert.Equal(t, "subnet-12345678", columns[12].Value(sampleResource))
	assert.Equal(t, "10.0.1.100", columns[13].Value(sampleResource))
	assert.Equal(t, "sg-12345678", columns[14].Value(sampleResource))
	assert.Equal(t, "/mnt/efs", columns[15].Value(sampleResource))
	assert.Equal(t, "1000", columns[16].Value(sampleResource))
}

// MockEFSCollector is a mock implementation of EFSCollector for testing
type MockEFSCollector struct{}

func (m *MockEFSCollector) Name() string {
	return "efs"
}

func (m *MockEFSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	return []Resource{
		{
			Category:    "efs",
			SubCategory: "FileSystem",
			Name:        "test-filesystem",
			Region:      region,
			ARN:         "fs-12345678",
			RawData: map[string]any{
				"Type":        "FileSystem",
				"Performance": "generalPurpose",
				"Throughput":  "bursting",
				"Encrypted":   "true",
				"Size":        "100GB",
			},
		},
		{
			Category:    "efs",
			SubCategory: "MountTarget",
			Name:        "test-mount-target",
			Region:      region,
			ARN:         "fsmt-12345678",
			RawData: map[string]any{
				"Type":          "MountTarget",
				"Subnet":        "subnet-12345",
				"IPAddress":     "10.0.1.100",
				"SecurityGroup": "sg-12345",
			},
		},
	}, nil
}

func (m *MockEFSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
	}
}

func (m *MockEFSCollector) ShouldSort() bool {
	return true
}

func TestMockEFSCollector_Collect(t *testing.T) {
	collector := &MockEFSCollector{}
	cfg := &aws.Config{}
	region := "us-east-1"

	resources, err := collector.Collect(context.Background(), cfg, region)

	assert.NoError(t, err)
	assert.NotEmpty(t, resources)
	assert.Equal(t, 2, len(resources))

	// Check filesystem resource
	fsResource := resources[0]
	assert.Equal(t, "efs", fsResource.Category)
	assert.Equal(t, "FileSystem", fsResource.SubCategory)
	assert.Equal(t, "test-filesystem", fsResource.Name)
	assert.Equal(t, region, fsResource.Region)
	assert.Equal(t, "fs-12345678", fsResource.ARN)

	// Check mount target resource
	mtResource := resources[1]
	assert.Equal(t, "efs", mtResource.Category)
	assert.Equal(t, "MountTarget", mtResource.SubCategory)
	assert.Equal(t, "test-mount-target", mtResource.Name)
	assert.Equal(t, region, mtResource.Region)
	assert.Equal(t, "fsmt-12345678", mtResource.ARN)
}
