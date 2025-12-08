package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewIAMUserGroupCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewIAMUserGroupCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestNewIAMUserGroupCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewIAMUserGroupCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestIAMUserGroupCollector_Basic(t *testing.T) {
	collector := &IAMUserGroupCollector{
		client: &iam.Client{},
	}
	assert.Equal(t, "iam_user_group", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestIAMUserGroupCollector_GetColumns(t *testing.T) {
	collector := &IAMUserGroupCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Path", "PasswordLastUsed", "CreateDate", "AttachedUsers", "AttachedPolicies",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Security",
		SubCategory:    "IAM",
		SubSubCategory: "User",
		Name:           "test-user",
		Region:         "Global",
		ARN:            "arn:aws:iam::123456789012:user/test-user",
		RawData: map[string]interface{}{
			"Path":             "/",
			"PasswordLastUsed": "2023-09-25T01:07:55Z",
			"CreateDate":       "2023-09-25T01:07:55Z",
			"AttachedUsers":    "",
			"AttachedPolicies": "AdministratorAccess",
		},
	}

	expectedValues := []string{
		"Security", "IAM", "User", "test-user", "Global",
		"arn:aws:iam::123456789012:user/test-user", "/", "2023-09-25T01:07:55Z", "2023-09-25T01:07:55Z", "", "AdministratorAccess",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
