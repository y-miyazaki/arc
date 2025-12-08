package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewGlueCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewGlueCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewGlueCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewGlueCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestGlueCollector_Basic(t *testing.T) {
	collector := &GlueCollector{
		clients: map[string]*glue.Client{},
	}
	assert.Equal(t, "glue", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestGlueCollector_GetColumns(t *testing.T) {
	collector := &GlueCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "Description", "RoleARN", "Timeout", "WorkerType",
		"NumberOfWorkers", "MaxRetries", "GlueVersion", "Language", "ScriptLocation",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Analytics",
		SubCategory:    "Glue",
		SubSubCategory: "Job",
		Name:           "test-job",
		Region:         "us-east-1",
		ARN:            "test-job",
		RawData: map[string]interface{}{
			"Description":     "Test Glue job",
			"RoleARN":         "arn:aws:iam::123456789012:role/GlueServiceRole",
			"Timeout":         "60",
			"WorkerType":      "G.1X",
			"NumberOfWorkers": "2",
			"MaxRetries":      "0",
			"GlueVersion":     "3.0",
			"Language":        "python",
			"ScriptLocation":  "s3://my-bucket/scripts/test.py",
		},
	}

	expectedValues := []string{
		"Analytics", "Glue", "Job", "test-job", "us-east-1",
		"test-job", "Test Glue job", "arn:aws:iam::123456789012:role/GlueServiceRole", "60", "G.1X",
		"2", "0", "3.0", "python", "s3://my-bucket/scripts/test.py",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
