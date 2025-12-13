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
	// CloudFront is a global service; the collector ensures a client for
	// the control-plane region (`helpers.CloudFrontRegion`) is present even
	// when the caller passes an empty regions slice.
	assert.NotEmpty(t, collector.clients)
	assert.Contains(t, collector.clients, helpers.CloudFrontRegion)
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

func TestCloudFrontCollector_OriginColumns(t *testing.T) {
	collector := &CloudFrontCollector{}
	columns := collector.GetColumns()

	// Build a map of header -> index for easy lookup
	idx := make(map[string]int)
	for i, c := range columns {
		idx[c.Header] = i
	}

	require.Contains(t, idx, "OriginId")
	require.Contains(t, idx, "DomainName")
	require.Contains(t, idx, "OriginPath")
	require.Contains(t, idx, "OriginType")
	require.Contains(t, idx, "OriginAccessControlId")
	require.Contains(t, idx, "ConnectionTimeout")
	require.Contains(t, idx, "ResponseTimeout")
	require.Contains(t, idx, "Config")

	originType := "s3"
	originConfig := "OAC=oac-123(oac-name) ConnectionTimeout=10s ResponseTimeout=20s"

	sampleOrigin := Resource{
		Category:     "CloudFront",
		SubCategory1: "Distribution",
		SubCategory2: "Origin",
		Name:         "test-distribution.cloudfront.net",
		Region:       "Global",
		RawData: map[string]interface{}{
			"ID":                    "E1A2B3C4D5F6G",
			"OriginId":              "origin-1",
			"DomainName":            "example.s3.amazonaws.com",
			"OriginPath":            "/images",
			"OriginType":            &originType,
			"OriginAccessControlId": "oac-123 (oac-name)",
			"ConnectionTimeout":     int32(10),
			"ResponseTimeout":       int32(20),
			"Config":                &originConfig,
		},
	}

	assert.Equal(t, "origin-1", columns[idx["OriginId"]].Value(sampleOrigin))
	assert.Equal(t, "example.s3.amazonaws.com", columns[idx["DomainName"]].Value(sampleOrigin))
	assert.Equal(t, "/images", columns[idx["OriginPath"]].Value(sampleOrigin))
	assert.Equal(t, "s3", columns[idx["OriginType"]].Value(sampleOrigin))
	assert.Equal(t, "oac-123 (oac-name)", columns[idx["OriginAccessControlId"]].Value(sampleOrigin))
	assert.Equal(t, "10", columns[idx["ConnectionTimeout"]].Value(sampleOrigin))
	assert.Equal(t, "20", columns[idx["ResponseTimeout"]].Value(sampleOrigin))
	assert.Equal(t, originConfig, columns[idx["Config"]].Value(sampleOrigin))
}

func TestCloudFrontCollector_BehaviorColumns(t *testing.T) {
	collector := &CloudFrontCollector{}
	columns := collector.GetColumns()

	// Build a map of header -> index for easy lookup
	idx := make(map[string]int)
	for i, c := range columns {
		idx[c.Header] = i
	}

	require.Contains(t, idx, "PathPattern")
	require.Contains(t, idx, "TargetOriginId")
	require.Contains(t, idx, "ViewerProtocolPolicy")
	require.Contains(t, idx, "CacheConfiguration")
	require.Contains(t, idx, "SmoothStreaming")
	require.Contains(t, idx, "RealtimeLogConfig")
	require.Contains(t, idx, "FunctionAssociations")
	require.Contains(t, idx, "Compress")

	realtimeArn := "arn:aws:logs:us-east-1:1234:realtime/log-config"

	sampleBehavior := Resource{
		Category:     "CloudFront",
		SubCategory1: "Distribution",
		SubCategory2: "Behavior",
		Name:         "test-distribution.cloudfront.net",
		Region:       "Global",
		RawData: map[string]interface{}{
			"ID":                   "E1A2B3C4D5F6G",
			"PathPattern":          "/img/*",
			"TargetOriginId":       "origin-1",
			"ViewerProtocolPolicy": "redirect-to-https",
			"CacheConfiguration":   []string{"CachePolicy=cp-1(cp-name)"},
			"SmoothStreaming":      true,
			"RealtimeLogConfig":    realtimeArn,
			"FunctionAssociations": []string{"FuncA=arn:aws:lambda:us-east-1:123:function:fnA"},
			"Compress":             true,
		},
	}

	assert.Equal(t, "/img/*", columns[idx["PathPattern"]].Value(sampleBehavior))
	assert.Equal(t, "origin-1", columns[idx["TargetOriginId"]].Value(sampleBehavior))
	assert.Equal(t, "redirect-to-https", columns[idx["ViewerProtocolPolicy"]].Value(sampleBehavior))
	// CacheConfiguration is a []string; Value will join it with newlines when converted
	assert.Equal(t, "CachePolicy=cp-1(cp-name)", columns[idx["CacheConfiguration"]].Value(sampleBehavior))
	assert.Equal(t, "true", columns[idx["SmoothStreaming"]].Value(sampleBehavior))
	assert.Equal(t, realtimeArn, columns[idx["RealtimeLogConfig"]].Value(sampleBehavior))
	assert.Equal(t, "FuncA=arn:aws:lambda:us-east-1:123:function:fnA", columns[idx["FunctionAssociations"]].Value(sampleBehavior))
	assert.Equal(t, "true", columns[idx["Compress"]].Value(sampleBehavior))
}
