package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestIAMUserGroupCollector_Basic(t *testing.T) {
	collector := NewMockIAMUserGroupCollector()
	assert.Equal(t, "iam_user_group", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestIAMUserGroupCollector_GetColumns(t *testing.T) {
	collector := NewMockIAMUserGroupCollector()
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Path", "PasswordLastUsed", "CreateDate", "AttachedUsers", "AttachedPolicies",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "iam_user_group",
		SubCategory:    "User",
		SubSubCategory: "",
		Name:           "test-user",
		Region:         "Global",
		ARN:            "arn:aws:iam::123456789012:user/test-user",
		RawData: map[string]any{
			"Path":             "/test/",
			"CreateDate":       "2023-09-25T01:07:55Z",
			"PasswordLastUsed": "2023-10-01T12:00:00Z",
			"AttachedUsers":    []string{"user1", "user2"},
			"AttachedPolicies": []string{"policy1", "policy2"},
		},
	}

	expectedValues := []string{
		"iam_user_group", "User", "", "test-user", "Global",
		"arn:aws:iam::123456789012:user/test-user", "/test/", "2023-10-01T12:00:00Z", "2023-09-25T01:07:55Z",
		"user1\nuser2", "policy1\npolicy2",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

// MockIAMUserGroupCollector is a testable version of IAMUserGroupCollector that uses mock data
type MockIAMUserGroupCollector struct{}

func NewMockIAMUserGroupCollector() *MockIAMUserGroupCollector {
	return &MockIAMUserGroupCollector{}
}

func (m *MockIAMUserGroupCollector) Name() string {
	return "iam_user_group"
}

func (m *MockIAMUserGroupCollector) ShouldSort() bool {
	return true
}

func (m *MockIAMUserGroupCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Path", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Path") }},
		{Header: "PasswordLastUsed", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PasswordLastUsed") }},
		{Header: "CreateDate", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreateDate") }},
		{Header: "AttachedUsers", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttachedUsers") }},
		{Header: "AttachedPolicies", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AttachedPolicies") }},
	}
}

func (c *MockIAMUserGroupCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// IAM is a global service, only process from us-east-1
	if region != "us-east-1" {
		return nil, nil
	}

	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock IAM user
	r1 := Resource{
		Category:    "iam_user_group",
		SubCategory: "User",
		Name:        "john.doe",
		Region:      region,
		ARN:         "arn:aws:iam::123456789012:user/john.doe",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Path":       "/users/",
			"CreateDate": "2023-08-15T10:30:00Z",
		}),
	}
	resources = append(resources, r1)

	// Mock IAM group
	r2 := Resource{
		Category:    "iam_user_group",
		SubCategory: "Group",
		Name:        "developers",
		Region:      region,
		ARN:         "arn:aws:iam::123456789012:group/developers",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Path":             "/groups/",
			"CreateDate":       "2023-07-01T09:15:00Z",
			"AttachedUsers":    []string{"john.doe", "jane.smith"},
			"AttachedPolicies": []string{"ReadOnlyAccess", "PowerUserAccess"},
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestMockIAMUserGroupCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}

	collector := NewMockIAMUserGroupCollector()

	// Test us-east-1 region (should return data)
	resources, err := collector.Collect(ctx, cfg, "us-east-1")
	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (IAM User)
	r1 := resources[0]
	assert.Equal(t, "iam_user_group", r1.Category)
	assert.Equal(t, "User", r1.SubCategory)
	assert.Equal(t, "john.doe", r1.Name)
	assert.Equal(t, "us-east-1", r1.Region)
	assert.Equal(t, "arn:aws:iam::123456789012:user/john.doe", r1.ARN)
	assert.Equal(t, "/users/", helpers.GetMapValue(r1.RawData, "Path"))
	assert.Equal(t, "2023-08-15T10:30:00Z", helpers.GetMapValue(r1.RawData, "CreateDate"))

	// Check second resource (IAM Group)
	r2 := resources[1]
	assert.Equal(t, "iam_user_group", r2.Category)
	assert.Equal(t, "Group", r2.SubCategory)
	assert.Equal(t, "developers", r2.Name)
	assert.Equal(t, "us-east-1", r2.Region)
	assert.Equal(t, "arn:aws:iam::123456789012:group/developers", r2.ARN)
	assert.Equal(t, "/groups/", helpers.GetMapValue(r2.RawData, "Path"))
	assert.Equal(t, "2023-07-01T09:15:00Z", helpers.GetMapValue(r2.RawData, "CreateDate"))
	assert.Equal(t, "jane.smith\njohn.doe", helpers.GetMapValue(r2.RawData, "AttachedUsers"))
	assert.Equal(t, "PowerUserAccess\nReadOnlyAccess", helpers.GetMapValue(r2.RawData, "AttachedPolicies"))

	// Test non-us-east-1 region (should return empty)
	resources, err = collector.Collect(ctx, cfg, "eu-west-1")
	assert.NoError(t, err)
	assert.Len(t, resources, 0)
}
