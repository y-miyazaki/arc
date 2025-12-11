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
		"Category", "SubCategory1", "Name", "Region", "ARN",
		"Versioning", "BucketABAC", "Encryption", "KMSKey", "AccessLogARN",
		"TransferAcceleration", "ObjectLock", "RequesterPays", "StaticWebsiteHosting",
		"PABBlockPublicACLs", "PABIgnorePublicACLs", "PABBlockPublicPolicy",
		"PABRestrictPublicBuckets", "ACL", "LifecycleRules", "CreationDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Storage",
		SubCategory1: "S3",
		SubCategory2: "Bucket",
		Name:         "test-bucket",
		Region:       "us-east-1",
		ARN:          "arn:aws:s3:::test-bucket",
		RawData: map[string]interface{}{
			"Versioning":               "Enabled",
			"BucketABAC":               "[Environment=Production Team=DevOps]",
			"Encryption":               "AES256",
			"KMSKey":                   "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			"AccessLogARN":             "arn:aws:s3:::log-bucket",
			"TransferAcceleration":     "Enabled",
			"ObjectLock":               "Enabled",
			"RequesterPays":            "Requester",
			"StaticWebsiteHosting":     "Enabled",
			"PABBlockPublicACLs":       "true",
			"PABIgnorePublicACLs":      "true",
			"PABBlockPublicPolicy":     "true",
			"PABRestrictPublicBuckets": "true",
			"ACL":                      "[CanonicalUser:abc123=FULL_CONTROL]",
			"LifecycleRules":           "2",
			"CreationDate":             "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"Storage", "S3", "test-bucket", "us-east-1", "arn:aws:s3:::test-bucket",
		"Enabled", "[Environment=Production Team=DevOps]", "AES256", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "arn:aws:s3:::log-bucket",
		"Enabled", "Enabled", "Requester", "Enabled",
		"true", "true", "true", "true", "[CanonicalUser:abc123=FULL_CONTROL]", "2", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
