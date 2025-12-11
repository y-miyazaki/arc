package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewIAMRoleCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewIAMRoleCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestNewIAMRoleCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewIAMRoleCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestIAMRoleCollector_Basic(t *testing.T) {
	collector := &IAMRoleCollector{
		client: &iam.Client{},
	}
	assert.Equal(t, "iam_role", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestIAMRoleCollector_GetColumns(t *testing.T) {
	collector := &IAMRoleCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region",
		"ARN", "Path", "AttachedPolicies", "PermissionsBoundary", "CreateDate", "LastUsedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Security",
		SubCategory1: "IAM",
		Name:         "test-role",
		Region:       "Global",
		ARN:          "arn:aws:iam::123456789012:role/test-role",
		RawData: map[string]interface{}{
			"Path":                "/",
			"AttachedPolicies":    "ReadOnlyAccess,PowerUserAccess",
			"PermissionsBoundary": "arn:aws:iam::123456789012:policy/boundary",
			"CreateDate":          "2023-09-25T01:07:55Z",
			"LastUsedDate":        "2023-09-26T10:30:00Z",
		},
	}

	expectedValues := []string{
		"Security", "IAM", "test-role", "Global",
		"arn:aws:iam::123456789012:role/test-role", "/", "ReadOnlyAccess,PowerUserAccess", "arn:aws:iam::123456789012:policy/boundary", "2023-09-25T01:07:55Z", "2023-09-26T10:30:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
