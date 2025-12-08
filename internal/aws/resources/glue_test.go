package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockGlueCollector is a testable version of GlueCollector that uses mock data
type MockGlueCollector struct{}

func NewMockGlueCollector() *MockGlueCollector {
	return &MockGlueCollector{}
}

func (c *MockGlueCollector) Name() string {
	return "glue"
}

func (c *MockGlueCollector) ShouldSort() bool {
	return true
}

func (c *MockGlueCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ID", Value: func(r Resource) string { return r.ARN }}, // Using ARN field for ID (Name in bash script)
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "RoleARN", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "RoleARN") }},
		{Header: "Timeout", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Timeout") }},
		{Header: "WorkerType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "WorkerType") }},
		{Header: "NumberOfWorkers", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NumberOfWorkers") }},
		{Header: "MaxRetries", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MaxRetries") }},
		{Header: "GlueVersion", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "GlueVersion") }},
		{Header: "Language", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Language") }},
		{Header: "ScriptLocation", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ScriptLocation") }},
	}
}

func (c *MockGlueCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Database
	r1 := Resource{
		Category:    "glue",
		SubCategory: "Database",
		Name:        "my-glue-database",
		Region:      region,
		ARN:         "my-glue-database",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description": "Database for ETL operations",
		}),
	}
	resources = append(resources, r1)

	// Mock Job
	r2 := Resource{
		Category:    "glue",
		SubCategory: "Job",
		Name:        "my-etl-job",
		Region:      region,
		ARN:         "my-etl-job",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description":     "ETL job for data processing",
			"RoleARN":         "arn:aws:iam::123456789012:role/GlueRole",
			"Timeout":         2880,
			"WorkerType":      "G.1X",
			"NumberOfWorkers": 2,
			"MaxRetries":      3,
			"GlueVersion":     "3.0",
			"Language":        "Python3",
			"ScriptLocation":  "s3://my-bucket/scripts/etl.py",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestGlueCollector_Basic(t *testing.T) {
	collector := &GlueCollector{}
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
		Category:       "glue",
		SubCategory:    "Job",
		SubSubCategory: "",
		Name:           "data-processing-job",
		Region:         "us-east-1",
		ARN:            "data-processing-job",
		RawData: map[string]any{
			"Description":     "ETL job for processing customer data",
			"RoleARN":         "arn:aws:iam::123456789012:role/GlueServiceRole",
			"Timeout":         "3600",
			"WorkerType":      "G.2X",
			"NumberOfWorkers": "4",
			"MaxRetries":      "2",
			"GlueVersion":     "4.0",
			"Language":        "Python3",
			"ScriptLocation":  "s3://my-scripts-bucket/etl/process_data.py",
		},
	}

	expectedValues := []string{
		"glue", "Job", "", "data-processing-job", "us-east-1",
		"data-processing-job", "ETL job for processing customer data",
		"arn:aws:iam::123456789012:role/GlueServiceRole", "3600", "G.2X",
		"4", "2", "4.0", "Python3", "s3://my-scripts-bucket/etl/process_data.py",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockGlueCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockGlueCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (Database)
	r1 := resources[0]
	assert.Equal(t, "glue", r1.Category)
	assert.Equal(t, "Database", r1.SubCategory)
	assert.Equal(t, "my-glue-database", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "my-glue-database", r1.ARN)
	assert.Equal(t, "Database for ETL operations", helpers.GetMapValue(r1.RawData, "Description"))

	// Check second resource (Job)
	r2 := resources[1]
	assert.Equal(t, "glue", r2.Category)
	assert.Equal(t, "Job", r2.SubCategory)
	assert.Equal(t, "my-etl-job", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "my-etl-job", r2.ARN)
	assert.Equal(t, "ETL job for data processing", helpers.GetMapValue(r2.RawData, "Description"))
	assert.Equal(t, "arn:aws:iam::123456789012:role/GlueRole", helpers.GetMapValue(r2.RawData, "RoleARN"))
	assert.Equal(t, "2880", helpers.GetMapValue(r2.RawData, "Timeout"))
	assert.Equal(t, "G.1X", helpers.GetMapValue(r2.RawData, "WorkerType"))
	assert.Equal(t, "2", helpers.GetMapValue(r2.RawData, "NumberOfWorkers"))
	assert.Equal(t, "3", helpers.GetMapValue(r2.RawData, "MaxRetries"))
	assert.Equal(t, "3.0", helpers.GetMapValue(r2.RawData, "GlueVersion"))
	assert.Equal(t, "Python3", helpers.GetMapValue(r2.RawData, "Language"))
	assert.Equal(t, "s3://my-bucket/scripts/etl.py", helpers.GetMapValue(r2.RawData, "ScriptLocation"))
}
