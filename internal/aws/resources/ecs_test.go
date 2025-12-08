package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestECSCollector_Name(t *testing.T) {
	collector := &ECSCollector{}
	assert.Equal(t, "ecs", collector.Name())
}

func TestECSCollector_ShouldSort(t *testing.T) {
	collector := &ECSCollector{}
	assert.False(t, collector.ShouldSort())
}

func TestECSCollector_GetColumns(t *testing.T) {
	collector := &ECSCollector{}
	columns := collector.GetColumns()

	assert.NotEmpty(t, columns)
	assert.Contains(t, columns[0].Header, "Category")
	assert.Contains(t, columns[1].Header, "SubCategory")
	assert.Contains(t, columns[2].Header, "SubSubCategory")
	assert.Contains(t, columns[3].Header, "Name")
	assert.Contains(t, columns[4].Header, "Region")

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "ecs",
		SubCategory:    "Service",
		SubSubCategory: "Fargate",
		Name:           "my-service",
		Region:         "us-east-1",
		ARN:            "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service",
		RawData: map[string]any{
			"RoleARN":         "arn:aws:iam::123456789012:role/ecs-service-role",
			"TaskDefinition":  "arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1",
			"LaunchType":      "FARGATE",
			"Status":          "ACTIVE",
			"CronSchedule":    "cron(0 2 * * ? *)",
			"Spec":            "FARGATE",
			"RuntimePlatform": "LINUX",
			"PortMappings":    "80:80,443:443",
			"Environment":     "ENV=prod,DEBUG=false",
		},
	}

	// Test each Value function
	assert.Equal(t, "ecs", columns[0].Value(sampleResource))
	assert.Equal(t, "Service", columns[1].Value(sampleResource))
	assert.Equal(t, "Fargate", columns[2].Value(sampleResource))
	assert.Equal(t, "my-service", columns[3].Value(sampleResource))
	assert.Equal(t, "us-east-1", columns[4].Value(sampleResource))
	assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service", columns[5].Value(sampleResource))
	assert.Equal(t, "arn:aws:iam::123456789012:role/ecs-service-role", columns[6].Value(sampleResource))
	assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:task-definition/my-task:1", columns[7].Value(sampleResource))
	assert.Equal(t, "FARGATE", columns[8].Value(sampleResource))
	assert.Equal(t, "ACTIVE", columns[9].Value(sampleResource))
	assert.Equal(t, "cron(0 2 * * ? *)", columns[10].Value(sampleResource))
	assert.Equal(t, "FARGATE", columns[11].Value(sampleResource))
	assert.Equal(t, "LINUX", columns[12].Value(sampleResource))
	assert.Equal(t, "80:80,443:443", columns[13].Value(sampleResource))
	assert.Equal(t, "ENV=prod,DEBUG=false", columns[14].Value(sampleResource))
}

// MockECSCollector is a mock implementation of ECSCollector for testing
type MockECSCollector struct{}

func (m *MockECSCollector) Name() string {
	return "ecs"
}

func (m *MockECSCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	return []Resource{
		{
			Category:    "ecs",
			SubCategory: "Cluster",
			Name:        "test-cluster",
			Region:      region,
			RawData: map[string]any{
				"ClusterName":                       "test-cluster",
				"Status":                            "ACTIVE",
				"RegisteredContainerInstancesCount": 2,
				"RunningTasksCount":                 5,
				"PendingTasksCount":                 0,
			},
		},
		{
			Category:    "ecs",
			SubCategory: "Service",
			Name:        "test-service",
			Region:      region,
			RawData: map[string]any{
				"ServiceName":    "test-service",
				"ClusterName":    "test-cluster",
				"Status":         "ACTIVE",
				"DesiredCount":   3,
				"RunningCount":   3,
				"TaskDefinition": "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
			},
		},
	}, nil
}

func (m *MockECSCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
	}
}

func (m *MockECSCollector) ShouldSort() bool {
	return false
}

func TestMockECSCollector_Collect(t *testing.T) {
	collector := &MockECSCollector{}
	cfg := &aws.Config{}
	region := "us-east-1"

	resources, err := collector.Collect(context.Background(), cfg, region)

	assert.NoError(t, err)
	assert.NotEmpty(t, resources)
	assert.Equal(t, 2, len(resources))

	// Check cluster resource
	clusterResource := resources[0]
	assert.Equal(t, "ecs", clusterResource.Category)
	assert.Equal(t, "Cluster", clusterResource.SubCategory)
	assert.Equal(t, "test-cluster", clusterResource.Name)
	assert.Equal(t, region, clusterResource.Region)

	// Check service resource
	serviceResource := resources[1]
	assert.Equal(t, "ecs", serviceResource.Category)
	assert.Equal(t, "Service", serviceResource.SubCategory)
	assert.Equal(t, "test-service", serviceResource.Name)
	assert.Equal(t, region, serviceResource.Region)
}
