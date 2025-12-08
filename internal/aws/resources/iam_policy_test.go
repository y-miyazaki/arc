package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewIAMPolicyCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewIAMPolicyCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestNewIAMPolicyCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewIAMPolicyCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestIAMPolicyCollector_Basic(t *testing.T) {
	collector := &IAMPolicyCollector{
		client: &iam.Client{},
	}
	assert.Equal(t, "iam_policy", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestIAMPolicyCollector_GetColumns(t *testing.T) {
	collector := &IAMPolicyCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Description", "Scope", "Path", "CreateDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Security",
		SubCategory:    "IAM",
		SubSubCategory: "Policy",
		Name:           "test-policy",
		Region:         "Global",
		ARN:            "arn:aws:iam::123456789012:policy/test-policy",
		RawData: map[string]interface{}{
			"Description": "Test IAM policy",
			"Scope":       "Local",
			"Path":        "/",
			"CreateDate":  "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Security", "IAM", "Policy", "test-policy", "Global",
		"arn:aws:iam::123456789012:policy/test-policy", "Test IAM policy", "Local", "/", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
