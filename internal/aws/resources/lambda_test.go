package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewLambdaCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewLambdaCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewLambdaCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewLambdaCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestLambdaCollector_Basic(t *testing.T) {
	collector := &LambdaCollector{
		clients: map[string]*lambda.Client{},
	}
	assert.Equal(t, "lambda", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestLambdaCollector_GetColumns(t *testing.T) {
	collector := &LambdaCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region",
		"ARN", "RoleARN", "Type", "Runtime", "Architecture",
		"MemorySize", "Timeout", "EnvVars", "LastModified",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Compute",
		SubCategory1: "Lambda",
		Name:         "test-function",
		Region:       "us-east-1",
		ARN:          "arn:aws:lambda:us-east-1:123456789012:function:test-function",
		RawData: map[string]interface{}{
			"RoleARN":      "arn:aws:iam::123456789012:role/lambda-role",
			"Type":         "Function",
			"Runtime":      "python3.9",
			"Architecture": "x86_64",
			"MemorySize":   "128",
			"Timeout":      "30",
			"EnvVars":      "KEY1=value1,KEY2=value2",
			"LastModified": "2023-09-25T01:07:55.000+0000",
		},
	}

	expectedValues := []string{
		"Compute", "Lambda", "test-function", "us-east-1",
		"arn:aws:lambda:us-east-1:123456789012:function:test-function", "arn:aws:iam::123456789012:role/lambda-role", "Function", "python3.9", "x86_64",
		"128", "30", "KEY1=value1,KEY2=value2", "2023-09-25T01:07:55.000+0000",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
