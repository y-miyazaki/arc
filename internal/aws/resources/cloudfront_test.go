package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCloudFrontCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewCloudFrontCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewCloudFrontCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewCloudFrontCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestCloudFrontCollector_Basic(t *testing.T) {
	collector := &CloudFrontCollector{
		clients: make(map[string]*cloudfront.Client),
	}
	assert.Equal(t, "cloudfront", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestCloudFrontCollector_Collect_NoClient(t *testing.T) {
	collector := &CloudFrontCollector{
		clients: make(map[string]*cloudfront.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-east-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestCloudFrontCollector_GetColumns(t *testing.T) {
	collector := &CloudFrontCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ID", "Description",
		"AlternateDomain", "Origin",
		"SSLCertificate", "SecurityPolicy", "SupportedHTTPVersions", "DefaultRootObject",
		"PriceClass", "WAF", "AccessLogDestinations",
		"OriginId", "DomainName", "OriginPath", "OriginType",
		"OriginAccessControlId", "OriginShield", "ConnectionTimeout", "ResponseTimeout",
		"Config",
		"PathPattern", "TargetOriginId", "ViewerProtocolPolicy",
		"CacheConfiguration",
		"SmoothStreaming", "RealtimeLogConfig", "FunctionAssociations",
		"Compress", "HTTPErrorCode", "ErrorCachingMinTTL", "CustomizeErrorResponse", "Status",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample Distribution resource
	sampleDistribution := Resource{
		Category:     "CloudFront",
		SubCategory1: "Distribution",
		SubCategory2: "",
		Name:         "test-distribution.cloudfront.net",
		Region:       "Global",
		ARN:          "",
		RawData: map[string]interface{}{
			"ID":                    "E1A2B3C4D5F6G",
			"Description":           "Test Distribution",
			"AlternateDomain":       "cdn.example.com",
			"Origin":                "example.s3.amazonaws.com",
			"SSLCertificate":        "arn:aws:acm:us-east-1:123456789012:certificate/test-cert",
			"SecurityPolicy":        "TLSv1.2_2021",
			"SupportedHTTPVersions": "http2and3",
			"DefaultRootObject":     "index.html",
			"PriceClass":            "PriceClass_100",
			"WAF":                   "test-waf",
			"AccessLogDestinations": "my-logs-bucket.s3.amazonaws.com/cloudfront",
			"Status":                "Deployed",
		},
	}

	expectedDistributionValues := []string{
		"CloudFront", "Distribution", "", "test-distribution.cloudfront.net", "Global", "E1A2B3C4D5F6G", "Test Distribution",
		"cdn.example.com", "example.s3.amazonaws.com",
		"arn:aws:acm:us-east-1:123456789012:certificate/test-cert", "TLSv1.2_2021",
		"http2and3", "index.html",
		"PriceClass_100", "test-waf",
		"my-logs-bucket.s3.amazonaws.com/cloudfront",
		"", "", "", "",
		"", "", "", "",
		"",
		"", "", "",
		"",
		"", "", "",
		"", "", "", "", "Deployed",
	}

	for i, column := range columns {
		assert.Equal(t, expectedDistributionValues[i], column.Value(sampleDistribution), "Column %d (%s) value mismatch", i, column.Header)
	}
}

func TestCloudFrontCollector_ErrorPageColumns(t *testing.T) {
	collector := &CloudFrontCollector{}
	columns := collector.GetColumns()

	// Build a map of header -> index for easy lookup
	idx := make(map[string]int)
	for i, c := range columns {
		idx[c.Header] = i
	}

	require.Contains(t, idx, "HTTPErrorCode")
	require.Contains(t, idx, "ErrorCachingMinTTL")
	require.Contains(t, idx, "CustomizeErrorResponse")

	sampleErrorPage := Resource{
		Category:     "CloudFront",
		SubCategory1: "Distribution",
		SubCategory2: "ErrorPage",
		Name:         "test-distribution.cloudfront.net",
		Region:       "Global",
		RawData: map[string]interface{}{
			"ID":                     "E1A2B3C4D5F6G",
			"HTTPErrorCode":          int32(404),
			"ErrorCachingMinTTL":     int64(60),
			"CustomizeErrorResponse": "ResponseCode=200 ResponsePagePath=/error.html",
			"Status":                 "Deployed",
		},
	}

	// Verify each ErrorPage-specific column returns the expected string
	assert.Equal(t, "404", columns[idx["HTTPErrorCode"]].Value(sampleErrorPage))
	assert.Equal(t, "60", columns[idx["ErrorCachingMinTTL"]].Value(sampleErrorPage))
	assert.Equal(t, "ResponseCode=200 ResponsePagePath=/error.html", columns[idx["CustomizeErrorResponse"]].Value(sampleErrorPage))
}
