package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestIAMRoleCollector_Basic(t *testing.T) {
	collector := NewMockIAMRoleCollector()
	assert.Equal(t, "iam_role", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestIAMRoleCollector_GetColumns(t *testing.T) {
	collector := NewMockIAMRoleCollector()
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Path", "AttachedPolicies", "CreateDate", "LastUsedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "iam_role",
		SubCategory:    "Role",
		SubSubCategory: "",
		Name:           "lambda-execution-role",
		Region:         "Global",
		ARN:            "arn:aws:iam::123456789012:role/lambda-execution-role",
		RawData: map[string]any{
			"Path":             "/service-role/",
			"CreateDate":       "2023-09-25T01:07:55Z",
			"LastUsedDate":     "2023-10-01T12:00:00Z",
			"AttachedPolicies": []string{"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"},
		},
	}

	expectedValues := []string{
		"iam_role", "Role", "", "lambda-execution-role", "Global",
		"arn:aws:iam::123456789012:role/lambda-execution-role", "/service-role/", "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole", "2023-09-25T01:07:55Z",
		"2023-10-01T12:00:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

// MockIAMRoleCollector is a testable version of IAMRoleCollector that uses mock data
type MockIAMRoleCollector struct{}

func NewMockIAMRoleCollector() *MockIAMRoleCollector {
	return &MockIAMRoleCollector{}
}

func (m *MockIAMRoleCollector) Name() string {
	return "iam_role"
}

func (m *MockIAMRoleCollector) ShouldSort() bool {
	return true
}

func (m *MockIAMRoleCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Path", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Path") }},
		{Header: "AttachedPolicies", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttachedPolicies") }},
		{Header: "CreateDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreateDate") }},
		{Header: "LastUsedDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "LastUsedDate") }},
	}
}
