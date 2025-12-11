package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewECSCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewECSCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.ebClients)
	assert.Len(t, collector.ebClients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewECSCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewECSCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.ebClients)
	assert.Len(t, collector.ebClients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestECSCollector_Basic(t *testing.T) {
	collector := &ECSCollector{
		clients:   map[string]*ecs.Client{},
		ebClients: map[string]*eventbridge.Client{},
	}
	assert.Equal(t, "ecs", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestECSCollector_GetColumns(t *testing.T) {
	collector := &ECSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ARN",
		"RoleARN", "TaskDefinition", "LaunchType", "Status", "CronSchedule",
		"Spec", "RuntimePlatform", "PortMappings", "Environment",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "ECS",
		SubCategory1: "Service",
		SubCategory2: "",
		Name:         "test-service",
		Region:       "us-east-1",
		ARN:          "arn:aws:ecs:us-east-1:123456789012:service/test-cluster/test-service",
		RawData: map[string]interface{}{
			"RoleARN":         "arn:aws:iam::123456789012:role/ecsTaskExecutionRole",
			"TaskDefinition":  "test-task-definition:1",
			"LaunchType":      "FARGATE",
			"Status":          "ACTIVE",
			"CronSchedule":    "cron(0 12 * * ? *)",
			"Spec":            "CPU: 256, Memory: 512",
			"RuntimePlatform": "LINUX/X86_64",
			"PortMappings":    "80/tcp",
			"Environment":     "KEY1=value1\nKEY2=value2",
		},
	}

	expectedValues := []string{
		"ECS", "Service", "", "test-service", "us-east-1", "arn:aws:ecs:us-east-1:123456789012:service/test-cluster/test-service",
		"arn:aws:iam::123456789012:role/ecsTaskExecutionRole", "test-task-definition:1", "FARGATE", "ACTIVE", "cron(0 12 * * ? *)",
		"CPU: 256, Memory: 512", "LINUX/X86_64", "80/tcp", "KEY1=value1\nKEY2=value2",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
