package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewBatchCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewBatchCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewBatchCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewBatchCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestBatchCollector_Basic(t *testing.T) {
	collector := &BatchCollector{
		clients: make(map[string]*batch.Client),
	}
	assert.Equal(t, "batch", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestBatchCollector_Collect_NoClient(t *testing.T) {
	collector := &BatchCollector{
		clients: make(map[string]*batch.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Batch client found for region")
}

func TestBatchCollector_GetColumns(t *testing.T) {
	collector := &BatchCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region", "ARN",
		"Priority", "Type", "JobRoleArn", "ExecutionRoleArn", "Image",
		"vCPU", "Memory", "CpuArchitecture", "OperatingSystemFamily", "Timeout", "JSON", "Status",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Batch",
		SubCategory1: "Job Queue",
		Name:         "test-queue",
		Region:       "us-east-1",
		ARN:          "arn:aws:batch:us-east-1:123456789012:job-queue/test-queue",
		RawData: map[string]interface{}{
			"Priority":              "1",
			"Type":                  "EC2",
			"JobRoleArn":            "arn:aws:iam::123456789012:role/BatchJobRole",
			"ExecutionRoleArn":      "arn:aws:iam::123456789012:role/BatchExecutionRole",
			"Image":                 "busybox",
			"vCPU":                  "1",
			"Memory":                "512",
			"CpuArchitecture":       "X86_64",
			"OperatingSystemFamily": "LINUX",
			"Timeout":               "3600",
			"JSON":                  "{}",
			"Status":                "ACTIVE",
		},
	}

	expectedValues := []string{
		"Batch", "Job Queue", "test-queue", "us-east-1", "arn:aws:batch:us-east-1:123456789012:job-queue/test-queue",
		"1", "EC2", "arn:aws:iam::123456789012:role/BatchJobRole", "arn:aws:iam::123456789012:role/BatchExecutionRole", "busybox",
		"1", "512", "X86_64", "LINUX", "3600", "{}", "ACTIVE",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
