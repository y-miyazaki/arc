package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockBatchCollector is a testable version of BatchCollector that uses mock data
type MockBatchCollector struct{}

func NewMockBatchCollector() *MockBatchCollector {
	return &MockBatchCollector{}
}

func (c *MockBatchCollector) Name() string {
	return "batch"
}

func (c *MockBatchCollector) ShouldSort() bool {
	return true
}

func (c *MockBatchCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Priority", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Priority") }},
		{Header: "Type", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Type") }},
		{Header: "JobRoleArn", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "JobRoleArn") }},
		{Header: "ExecutionRoleArn", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ExecutionRoleArn") }},
		{Header: "Image", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Image") }},
		{Header: "vCPU", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "vCPU") }},
		{Header: "Memory", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Memory") }},
		{Header: "CpuArchitecture", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CpuArchitecture") }},
		{Header: "OperatingSystemFamily", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "OperatingSystemFamily") }},
		{Header: "Timeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Timeout") }},
		{Header: "JSON", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "JSON") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
	}
}

func (c *MockBatchCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Job Queue
	r1 := Resource{
		Category:    "batch",
		SubCategory: "JobQueue",
		Name:        "my-job-queue",
		Region:      region,
		ARN:         "arn:aws:batch:us-east-1:123456789012:job-queue/my-job-queue",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Priority": "10",
			"Status":   "ENABLED",
			"JSON":     `{"jobQueueName":"my-job-queue","priority":10}`,
		}),
	}
	resources = append(resources, r1)

	// Mock Compute Environment
	r2 := Resource{
		Category:    "batch",
		SubCategory: "ComputeEnvironment",
		Name:        "my-compute-env",
		Region:      region,
		ARN:         "arn:aws:batch:us-east-1:123456789012:compute-environment/my-compute-env",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Type":   "MANAGED",
			"Status": "ENABLED",
			"JSON":   `{"computeEnvironmentName":"my-compute-env","type":"MANAGED"}`,
		}),
	}
	resources = append(resources, r2)

	// Mock Job Definition
	r3 := Resource{
		Category:    "batch",
		SubCategory: "JobDefinition",
		Name:        "my-job-def:1",
		Region:      region,
		ARN:         "arn:aws:batch:us-east-1:123456789012:job-definition/my-job-def:1",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Type":                  "container",
			"JobRoleArn":            "arn:aws:iam::123456789012:role/BatchJobRole",
			"ExecutionRoleArn":      "arn:aws:iam::123456789012:role/BatchExecutionRole",
			"Image":                 "busybox:latest",
			"vCPU":                  "1",
			"Memory":                "512",
			"CpuArchitecture":       "X86_64",
			"OperatingSystemFamily": "LINUX",
			"Timeout":               "3600",
			"JSON":                  `{"jobDefinitionName":"my-job-def","revision":1}`,
			"Status":                "ACTIVE",
		}),
	}
	resources = append(resources, r3)

	return resources, nil
}

func TestBatchCollector_Basic(t *testing.T) {
	collector := &BatchCollector{}
	assert.Equal(t, "batch", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestBatchCollector_GetColumns(t *testing.T) {
	collector := &BatchCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Priority", "Type", "JobRoleArn", "ExecutionRoleArn",
		"Image", "vCPU", "Memory", "CpuArchitecture", "OperatingSystemFamily",
		"Timeout", "JSON", "Status",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "Compute",
		SubCategory:    "Batch",
		SubSubCategory: "JobDefinition",
		Name:           "my-job-def:1",
		Region:         "us-east-1",
		ARN:            "arn:aws:batch:us-east-1:123456789012:job-definition/my-job-def:1",
		RawData: map[string]any{
			"Priority":              "10",
			"Type":                  "container",
			"JobRoleArn":            "arn:aws:iam::123456789012:role/JobRole",
			"ExecutionRoleArn":      "arn:aws:iam::123456789012:role/ExecutionRole",
			"Image":                 "busybox:latest",
			"vCPU":                  "1",
			"Memory":                "512",
			"CpuArchitecture":       "X86_64",
			"OperatingSystemFamily": "LINUX",
			"Timeout":               "3600",
			"JSON":                  `{"test":"data"}`,
			"Status":                "ACTIVE",
		},
	}

	expectedValues := []string{
		"Compute", "Batch", "JobDefinition", "my-job-def:1", "us-east-1",
		"arn:aws:batch:us-east-1:123456789012:job-definition/my-job-def:1",
		"10", "container", "arn:aws:iam::123456789012:role/JobRole",
		"arn:aws:iam::123456789012:role/ExecutionRole", "busybox:latest",
		"1", "512", "X86_64", "LINUX", "3600", `{"test":"data"}`, "ACTIVE",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockBatchCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockBatchCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 3)

	// Check first resource (Job Queue)
	r1 := resources[0]
	assert.Equal(t, "batch", r1.Category)
	assert.Equal(t, "JobQueue", r1.SubCategory)
	assert.Equal(t, "my-job-queue", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:batch:us-east-1:123456789012:job-queue/my-job-queue", r1.ARN)
	assert.Equal(t, "10", helpers.GetMapValue(r1.RawData, "Priority"))
	assert.Equal(t, "ENABLED", helpers.GetMapValue(r1.RawData, "Status"))

	// Check second resource (Compute Environment)
	r2 := resources[1]
	assert.Equal(t, "batch", r2.Category)
	assert.Equal(t, "ComputeEnvironment", r2.SubCategory)
	assert.Equal(t, "my-compute-env", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:batch:us-east-1:123456789012:compute-environment/my-compute-env", r2.ARN)
	assert.Equal(t, "MANAGED", helpers.GetMapValue(r2.RawData, "Type"))
	assert.Equal(t, "ENABLED", helpers.GetMapValue(r2.RawData, "Status"))

	// Check third resource (Job Definition)
	r3 := resources[2]
	assert.Equal(t, "batch", r3.Category)
	assert.Equal(t, "JobDefinition", r3.SubCategory)
	assert.Equal(t, "my-job-def:1", r3.Name)
	assert.Equal(t, region, r3.Region)
	assert.Equal(t, "arn:aws:batch:us-east-1:123456789012:job-definition/my-job-def:1", r3.ARN)
	assert.Equal(t, "container", helpers.GetMapValue(r3.RawData, "Type"))
	assert.Equal(t, "arn:aws:iam::123456789012:role/BatchJobRole", helpers.GetMapValue(r3.RawData, "JobRoleArn"))
	assert.Equal(t, "arn:aws:iam::123456789012:role/BatchExecutionRole", helpers.GetMapValue(r3.RawData, "ExecutionRoleArn"))
	assert.Equal(t, "busybox:latest", helpers.GetMapValue(r3.RawData, "Image"))
	assert.Equal(t, "1", helpers.GetMapValue(r3.RawData, "vCPU"))
	assert.Equal(t, "512", helpers.GetMapValue(r3.RawData, "Memory"))
	assert.Equal(t, "X86_64", helpers.GetMapValue(r3.RawData, "CpuArchitecture"))
	assert.Equal(t, "LINUX", helpers.GetMapValue(r3.RawData, "OperatingSystemFamily"))
	assert.Equal(t, "3600", helpers.GetMapValue(r3.RawData, "Timeout"))
	assert.Equal(t, `{"jobDefinitionName":"my-job-def","revision":1}`, helpers.GetMapValue(r3.RawData, "JSON"))
	assert.Equal(t, "ACTIVE", helpers.GetMapValue(r3.RawData, "Status"))
}
