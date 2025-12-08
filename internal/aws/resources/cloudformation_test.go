package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestCloudFormationCollector_Name(t *testing.T) {
	collector := &CloudFormationCollector{}
	assert.Equal(t, "cloudformation", collector.Name())
}

func TestCloudFormationCollector_ShouldSort(t *testing.T) {
	collector := &CloudFormationCollector{}
	assert.True(t, collector.ShouldSort())
}

func TestCloudFormationCollector_GetColumns(t *testing.T) {
	collector := &CloudFormationCollector{}
	columns := collector.GetColumns()

	assert.NotEmpty(t, columns)
	assert.Contains(t, columns[0].Header, "Category")
	assert.Contains(t, columns[1].Header, "SubCategory")
	assert.Contains(t, columns[2].Header, "SubSubCategory")
	assert.Contains(t, columns[3].Header, "Name")
	assert.Contains(t, columns[4].Header, "Region")
	assert.Contains(t, columns[5].Header, "ARN")
	assert.Contains(t, columns[6].Header, "Description")

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "cloudformation",
		SubCategory:    "Stack",
		SubSubCategory: "",
		Name:           "my-stack",
		Region:         "us-east-1",
		ARN:            "arn:aws:cloudformation:us-east-1:123456789012:stack/my-stack/12345678-1234-1234-1234-123456789012",
		RawData: map[string]any{
			"Description": "My CloudFormation stack for testing",
			"Type":        "Stack",
			"Outputs":     "WebsiteURL=https://example.com,DatabaseEndpoint=db.example.com:5432",
			"Parameters":  "Environment=prod,InstanceType=t2.micro",
			"Resources":   "12",
			"CreatedDate": "2023-01-15T10:30:00Z",
			"UpdatedDate": "2023-01-20T14:45:00Z",
			"DriftStatus": "IN_SYNC",
			"Status":      "CREATE_COMPLETE",
		},
	}

	// Test each Value function
	assert.Equal(t, "cloudformation", columns[0].Value(sampleResource))
	assert.Equal(t, "Stack", columns[1].Value(sampleResource))
	assert.Equal(t, "", columns[2].Value(sampleResource))
	assert.Equal(t, "my-stack", columns[3].Value(sampleResource))
	assert.Equal(t, "us-east-1", columns[4].Value(sampleResource))
	assert.Equal(t, "arn:aws:cloudformation:us-east-1:123456789012:stack/my-stack/12345678-1234-1234-1234-123456789012", columns[5].Value(sampleResource))
	assert.Equal(t, "My CloudFormation stack for testing", columns[6].Value(sampleResource))
	assert.Equal(t, "Stack", columns[7].Value(sampleResource))
	assert.Equal(t, "WebsiteURL=https://example.com,DatabaseEndpoint=db.example.com:5432", columns[8].Value(sampleResource))
	assert.Equal(t, "Environment=prod,InstanceType=t2.micro", columns[9].Value(sampleResource))
	assert.Equal(t, "12", columns[10].Value(sampleResource))
	assert.Equal(t, "CREATE_COMPLETE", columns[11].Value(sampleResource))
	assert.Equal(t, "IN_SYNC", columns[12].Value(sampleResource))
	assert.Equal(t, "2023-01-15T10:30:00Z", columns[13].Value(sampleResource))
	assert.Equal(t, "2023-01-20T14:45:00Z", columns[14].Value(sampleResource))
}

// MockCloudFormationCollector is a mock implementation of CloudFormationCollector for testing
type MockCloudFormationCollector struct{}

func (m *MockCloudFormationCollector) Name() string {
	return "cloudformation"
}

func (m *MockCloudFormationCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	return []Resource{
		{
			Category:    "cloudformation",
			SubCategory: "Stack",
			Name:        "test-stack",
			Region:      region,
			ARN:         "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/12345678-1234-1234-1234-123456789012",
			RawData: map[string]any{
				"Description": "Test CloudFormation stack",
				"Type":        "Stack",
				"Outputs":     "2",
				"Parameters":  "1",
				"Resources":   "5",
				"CreatedDate": "2023-01-01T00:00:00Z",
				"UpdatedDate": "2023-01-02T00:00:00Z",
				"DriftStatus": "IN_SYNC",
				"Status":      "CREATE_COMPLETE",
			},
		},
		{
			Category:    "cloudformation",
			SubCategory: "StackSet",
			Name:        "test-stack-set",
			Region:      region,
			ARN:         "arn:aws:cloudformation:us-east-1:123456789012:stackset/test-stack-set:12345678-1234-1234-1234-123456789012",
			RawData: map[string]any{
				"Description": "Test CloudFormation stack set",
				"Type":        "StackSet",
				"Status":      "ACTIVE",
			},
		},
	}, nil
}

func (m *MockCloudFormationCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
	}
}

func (m *MockCloudFormationCollector) ShouldSort() bool {
	return true
}

func TestMockCloudFormationCollector_Collect(t *testing.T) {
	collector := &MockCloudFormationCollector{}
	cfg := &aws.Config{}
	region := "us-east-1"

	resources, err := collector.Collect(context.Background(), cfg, region)

	assert.NoError(t, err)
	assert.NotEmpty(t, resources)
	assert.Equal(t, 2, len(resources))

	// Check stack resource
	stackResource := resources[0]
	assert.Equal(t, "cloudformation", stackResource.Category)
	assert.Equal(t, "Stack", stackResource.SubCategory)
	assert.Equal(t, "test-stack", stackResource.Name)
	assert.Equal(t, region, stackResource.Region)
	assert.Contains(t, stackResource.ARN, "test-stack")

	// Check stack set resource
	stackSetResource := resources[1]
	assert.Equal(t, "cloudformation", stackSetResource.Category)
	assert.Equal(t, "StackSet", stackSetResource.SubCategory)
	assert.Equal(t, "test-stack-set", stackSetResource.Name)
	assert.Equal(t, region, stackSetResource.Region)
	assert.Contains(t, stackSetResource.ARN, "test-stack-set")
}
