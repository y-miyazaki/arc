package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewS3BucketCollector(t *testing.T) {
	t.Parallel()

	cfg := &aws.Config{
		Region: "us-east-1",
	}

	tests := []struct {
		name    string
		regions []string
		wantLen int
	}{
		{name: "creates clients for each region", regions: []string{"us-east-1", "eu-west-1"}, wantLen: 2},
		{name: "empty regions", regions: []string{}, wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nameResolver, err := helpers.NewNameResolver(cfg, tt.regions)
			require.NoError(t, err)

			collector, err := NewS3BucketCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.NotNil(t, collector.client)
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestS3BucketCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "s3_bucket", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &S3BucketCollector{
				client: &s3.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestS3BucketCollector_GetColumns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		resource    Resource
		wantHeaders []string
		wantValues  []string
	}{
		{
			name: "headers and sample values",
			resource: Resource{
				Category:     "Storage",
				SubCategory1: "S3",
				SubCategory2: "Bucket",
				Name:         "test-bucket",
				Region:       "us-east-1",
				ARN:          "arn:aws:s3:::test-bucket",
				RawData: map[string]any{
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
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"Versioning", "BucketABAC", "Encryption", "KMSKey", "AccessLogARN",
				"TransferAcceleration", "ObjectLock", "RequesterPays", "StaticWebsiteHosting",
				"PABBlockPublicACLs", "PABIgnorePublicACLs", "PABBlockPublicPolicy",
				"PABRestrictPublicBuckets", "ACL", "LifecycleRules", "CreationDate",
			},
			wantValues: []string{
				"Storage", "S3", "test-bucket", "us-east-1", "arn:aws:s3:::test-bucket",
				"Enabled", "[Environment=Production Team=DevOps]", "AES256", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "arn:aws:s3:::log-bucket",
				"Enabled", "Enabled", "Requester", "Enabled",
				"true", "true", "true", "true", "[CanonicalUser:abc123=FULL_CONTROL]", "2", "2023-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &S3BucketCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
