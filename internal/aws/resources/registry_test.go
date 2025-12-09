package resources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockCollector is a mock implementation of the Collector interface for testing
type MockCollector struct {
	name        string
	shouldSort  bool
	columns     []Column
	collectFunc func(ctx context.Context, region string) ([]Resource, error)
}

func NewMockCollector(name string, shouldSort bool) *MockCollector {
	return &MockCollector{
		name:       name,
		shouldSort: shouldSort,
		columns: []Column{
			{Header: "Category", Value: func(r Resource) string { return r.Category }},
			{Header: "Name", Value: func(r Resource) string { return r.Name }},
			{Header: "Region", Value: func(r Resource) string { return r.Region }},
		},
		collectFunc: func(ctx context.Context, region string) ([]Resource, error) {
			return []Resource{
				{
					Category: "test",
					Name:     "test-resource",
					Region:   region,
					RawData: helpers.NormalizeRawData(map[string]any{
						"Status": "active",
					}),
				},
			}, nil
		},
	}
}

func (m *MockCollector) Name() string {
	return m.name
}

func (m *MockCollector) ShouldSort() bool {
	return m.shouldSort
}

func (m *MockCollector) GetColumns() []Column {
	return m.columns
}

func (m *MockCollector) Collect(ctx context.Context, region string) ([]Resource, error) {
	return m.collectFunc(ctx, region)
}

func TestNewResource(t *testing.T) {
	input := &ResourceInput{
		Category:    "test-category",
		SubCategory: "test-subcategory",
		Name:        "test-name",
		Region:      "us-east-1",
		ARN:         "arn:aws:test:us-east-1:123456789012:test/test-name",
		RawData: map[string]any{
			"Status":      "active",
			"CreatedDate": "2023-01-01T00:00:00Z",
			"Count":       42,
		},
	}

	resource := NewResource(input)

	assert.Equal(t, "test-category", resource.Category)
	assert.Equal(t, "test-subcategory", resource.SubCategory)
	assert.Equal(t, "", resource.SubSubCategory) // empty string for nil input
	assert.Equal(t, "test-name", resource.Name)
	assert.Equal(t, "us-east-1", resource.Region)
	assert.Equal(t, "arn:aws:test:us-east-1:123456789012:test/test-name", resource.ARN)

	// Check that RawData is normalized
	assert.Equal(t, "active", resource.RawData["Status"])
	assert.Equal(t, "2023-01-01T00:00:00Z", resource.RawData["CreatedDate"])
	assert.Equal(t, "42", resource.RawData["Count"]) // should be string
}

func TestNewResource_WithNilValues(t *testing.T) {
	input := &ResourceInput{
		Category: "test-category",
		Name:     "test-name",
		Region:   "us-east-1",
		RawData:  map[string]any{},
	}
	// Explicitly set nil values
	input.SubCategory = nil
	input.SubSubCategory = nil
	input.ARN = nil

	resource := NewResource(input)

	assert.Equal(t, "test-category", resource.Category)
	assert.Equal(t, "", resource.SubCategory)    // empty string for nil with default ""
	assert.Equal(t, "", resource.SubSubCategory) // empty string for nil with default ""
	assert.Equal(t, "test-name", resource.Name)
	assert.Equal(t, "us-east-1", resource.Region)
	assert.Equal(t, "", resource.ARN) // empty string for nil when default is empty
}

func TestRegister(t *testing.T) {
	// Clear the registry before test
	originalCollectors := make(map[string]Collector)
	for k, v := range collectors {
		originalCollectors[k] = v
	}
	collectors = make(map[string]Collector)
	defer func() {
		collectors = originalCollectors
	}()

	// Test registering a collector
	mockCollector := NewMockCollector("test-collector", true)
	Register("test", mockCollector)

	assert.Contains(t, collectors, "test")
	assert.Equal(t, mockCollector, collectors["test"])
}

func TestGetCollectors(t *testing.T) {
	// Clear the registry before test
	originalCollectors := make(map[string]Collector)
	for k, v := range collectors {
		originalCollectors[k] = v
	}
	collectors = make(map[string]Collector)
	defer func() {
		collectors = originalCollectors
	}()

	// Register some test collectors
	mockCollector1 := NewMockCollector("collector1", true)
	mockCollector2 := NewMockCollector("collector2", false)

	Register("test1", mockCollector1)
	Register("test2", mockCollector2)

	result := GetCollectors()

	assert.Len(t, result, 2)
	assert.Contains(t, result, "test1")
	assert.Contains(t, result, "test2")
	assert.Equal(t, mockCollector1, result["test1"])
	assert.Equal(t, mockCollector2, result["test2"])
}

func TestMockCollector_Name(t *testing.T) {
	collector := NewMockCollector("test-name", true)
	assert.Equal(t, "test-name", collector.Name())
}

func TestMockCollector_ShouldSort(t *testing.T) {
	collector := NewMockCollector("test", true)
	assert.True(t, collector.ShouldSort())

	collector2 := NewMockCollector("test", false)
	assert.False(t, collector2.ShouldSort())
}

func TestMockCollector_GetColumns(t *testing.T) {
	collector := NewMockCollector("test", true)
	columns := collector.GetColumns()

	expectedHeaders := []string{"Category", "Name", "Region"}
	assert.Len(t, columns, len(expectedHeaders))
	for i, header := range expectedHeaders {
		assert.Equal(t, header, columns[i].Header)
	}
}

func TestMockCollector_Collect(t *testing.T) {
	collector := NewMockCollector("test", true)

	ctx := context.Background()
	region := "us-west-2"

	resources, err := collector.Collect(ctx, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 1)

	resource := resources[0]
	assert.Equal(t, "test", resource.Category)
	assert.Equal(t, "test-resource", resource.Name)
	assert.Equal(t, region, resource.Region)
	assert.Equal(t, "active", resource.RawData["Status"])
}
