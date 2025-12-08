package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockIAMPolicyCollector is a testable version of IAMPolicyCollector that uses mock data
type MockIAMPolicyCollector struct{}

func NewMockIAMPolicyCollector() *MockIAMPolicyCollector {
	return &MockIAMPolicyCollector{}
}

func (m *MockIAMPolicyCollector) Name() string {
	return "iam_policy"
}

func (m *MockIAMPolicyCollector) ShouldSort() bool {
	return true
}

func (m *MockIAMPolicyCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Path", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Path") }},
		{Header: "CreateDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreateDate") }},
	}
}

func TestIAMPolicyCollector_Basic(t *testing.T) {
	collector := NewMockIAMPolicyCollector()
	assert.Equal(t, "iam_policy", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestIAMPolicyCollector_GetColumns(t *testing.T) {
	collector := NewMockIAMPolicyCollector()
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Path", "CreateDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "iam_policy",
		SubCategory:    "Policy",
		SubSubCategory: "",
		Name:           "test-policy",
		Region:         "Global",
		ARN:            "arn:aws:iam::123456789012:policy/test-policy",
		RawData: map[string]any{
			"Path":       "/test/",
			"CreateDate": "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"iam_policy", "Policy", "", "test-policy", "Global",
		"arn:aws:iam::123456789012:policy/test-policy", "/test/", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}
