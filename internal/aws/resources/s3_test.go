package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewS3Collector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewS3Collector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestNewS3Collector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewS3Collector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.nameResolver)
}

func TestS3Collector_Basic(t *testing.T) {
	collector := &S3Collector{
		client: &s3.Client{},
	}
	assert.Equal(t, "s3", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestS3Collector_GetColumns(t *testing.T) {
	collector := &S3Collector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"Encryption", "Versioning", "PABBlockPublicACLs", "PABIgnorePublicACLs", "PABBlockPublicPolicy",
		"PABRestrictPublicBuckets", "AccessLogARN", "LifecycleRules", "CreationDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Storage",
		SubCategory:    "S3",
		SubSubCategory: "Bucket",
		Name:           "test-bucket",
		Region:         "us-east-1",
		ARN:            "arn:aws:s3:::test-bucket",
		RawData: map[string]interface{}{
			"Encryption":               "AES256",
			"Versioning":               "Enabled",
			"PABBlockPublicACLs":       "true",
			"PABIgnorePublicACLs":      "true",
			"PABBlockPublicPolicy":     "true",
			"PABRestrictPublicBuckets": "true",
			"AccessLogARN":             "arn:aws:s3:::log-bucket",
			"LifecycleRules":           "2",
			"CreationDate":             "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Storage", "S3", "Bucket", "test-bucket", "us-east-1", "arn:aws:s3:::test-bucket",
		"AES256", "Enabled", "true", "true", "true", "true", "arn:aws:s3:::log-bucket", "2", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
