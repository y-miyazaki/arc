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
	t.Parallel()

	cfg := &aws.Config{
		Region: "us-east-1",
	}

	tests := []struct {
		name                 string
		regions              []string
		wantLen              int
		wantCloudFrontRegion bool
	}{
		{name: "creates clients for each region", regions: []string{"us-east-1", "eu-west-1"}, wantLen: 2},
		{name: "empty regions still has global client", regions: []string{}, wantLen: 0, wantCloudFrontRegion: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nameResolver, err := helpers.NewNameResolver(cfg, tt.regions)
			require.NoError(t, err)

			collector, err := NewCloudFrontCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			if tt.wantCloudFrontRegion {
				assert.NotEmpty(t, collector.clients)
				assert.Contains(t, collector.clients, helpers.CloudFrontRegion)
			} else {
				assert.Len(t, collector.clients, tt.wantLen)
				for _, region := range tt.regions {
					assert.Contains(t, collector.clients, region)
				}
			}
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestCloudFrontCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "cloudfront", wantSort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &CloudFrontCollector{
				clients: make(map[string]*cloudfront.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestCloudFrontCollector_Collect_NoClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		region       string
		wantContains string
	}{
		{name: "missing client returns error", region: "us-east-1", wantContains: "no client found for region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &CloudFrontCollector{
				clients: make(map[string]*cloudfront.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestCloudFrontCollector_GetColumns(t *testing.T) {
	t.Parallel()

	originType := "s3"
	originConfig := "OAC=oac-123(oac-name) ConnectionTimeout=10s ResponseTimeout=20s"
	realtimeArn := "arn:aws:logs:us-east-1:1234:realtime/log-config"

	tests := []struct {
		name         string
		resource     Resource
		wantHeaders  []string
		wantValues   []string
		wantByHeader map[string]string
	}{
		{
			name: "distribution headers and sample values",
			resource: Resource{
				Category:     "CloudFront",
				SubCategory1: "Distribution",
				Name:         "test-distribution.cloudfront.net",
				Region:       "Global",
				RawData: map[string]any{
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
			},
			wantHeaders: []string{
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
			},
			wantValues: []string{
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
			},
		},
		{
			name: "error page columns",
			resource: Resource{
				Category:     "CloudFront",
				SubCategory1: "Distribution",
				SubCategory2: "ErrorPage",
				Name:         "test-distribution.cloudfront.net",
				Region:       "Global",
				RawData: map[string]any{
					"ID":                     "E1A2B3C4D5F6G",
					"HTTPErrorCode":          int32(404),
					"ErrorCachingMinTTL":     int64(60),
					"CustomizeErrorResponse": "ResponseCode=200 ResponsePagePath=/error.html",
					"Status":                 "Deployed",
				},
			},
			wantByHeader: map[string]string{
				"HTTPErrorCode":          "404",
				"ErrorCachingMinTTL":     "60",
				"CustomizeErrorResponse": "ResponseCode=200 ResponsePagePath=/error.html",
			},
		},
		{
			name: "origin columns",
			resource: Resource{
				Category:     "CloudFront",
				SubCategory1: "Distribution",
				SubCategory2: "Origin",
				Name:         "test-distribution.cloudfront.net",
				Region:       "Global",
				RawData: map[string]any{
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
			},
			wantByHeader: map[string]string{
				"OriginId":              "origin-1",
				"DomainName":            "example.s3.amazonaws.com",
				"OriginPath":            "/images",
				"OriginType":            "s3",
				"OriginAccessControlId": "oac-123 (oac-name)",
				"ConnectionTimeout":     "10",
				"ResponseTimeout":       "20",
				"Config":                originConfig,
			},
		},
		{
			name: "behavior columns",
			resource: Resource{
				Category:     "CloudFront",
				SubCategory1: "Distribution",
				SubCategory2: "Behavior",
				Name:         "test-distribution.cloudfront.net",
				Region:       "Global",
				RawData: map[string]any{
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
			},
			wantByHeader: map[string]string{
				"PathPattern":          "/img/*",
				"TargetOriginId":       "origin-1",
				"ViewerProtocolPolicy": "redirect-to-https",
				"CacheConfiguration":   "CachePolicy=cp-1(cp-name)",
				"SmoothStreaming":      "true",
				"RealtimeLogConfig":    realtimeArn,
				"FunctionAssociations": "FuncA=arn:aws:lambda:us-east-1:123:function:fnA",
				"Compress":             "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			columns := (&CloudFrontCollector{}).GetColumns()
			idx := make(map[string]int, len(columns))
			for i, column := range columns {
				idx[column.Header] = i
			}
			if tt.wantHeaders != nil {
				require.Len(t, columns, len(tt.wantHeaders))
				for i, column := range columns {
					assert.Equal(t, tt.wantHeaders[i], column.Header)
					assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
				}
				return
			}
			for header, want := range tt.wantByHeader {
				require.Contains(t, idx, header)
				assert.Equal(t, want, columns[idx[header]].Value(tt.resource))
			}
		})
	}
}
