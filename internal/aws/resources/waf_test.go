package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewWAFCollector(t *testing.T) {
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

			collector, err := NewWAFCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.Len(t, collector.wafClient, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.wafClient, region)
			}
			assert.NotNil(t, collector.cfClient)
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestWAFCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "waf", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &WAFCollector{
				wafClient: make(map[string]*wafv2.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestWAFCollector_GetColumns(t *testing.T) {
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
				Category:     "Security",
				SubCategory1: "WAF",
				Name:         "test-web-acl",
				Region:       "us-east-1",
				ARN:          "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/test-web-acl/12345678-1234-1234-1234-123456789012",
				RawData: map[string]any{
					"Description":         "Test WebACL",
					"Scope":               "REGIONAL",
					"Rules":               []string{"Rule1", "Rule2"},
					"AssociatedResources": []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890abcdef"},
					"Logging":             []string{"arn:aws:s3:::aws-waf-logs-test"},
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"Description", "Scope", "Rules", "AssociatedResources", "Logging",
			},
			wantValues: []string{
				"Security", "WAF", "test-web-acl", "us-east-1", "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/test-web-acl/12345678-1234-1234-1234-123456789012",
				"Test WebACL", "REGIONAL", "Rule1\nRule2", "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890abcdef", "arn:aws:s3:::aws-waf-logs-test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &WAFCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}

func TestWAFCollector_Collect_NoClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		region       string
		wantContains string
	}{
		{name: "missing client returns error", region: "us-west-2", wantContains: "no client found for region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &WAFCollector{wafClient: make(map[string]*wafv2.Client)}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestWAFCollector_Collect_RegionalCollectScopeError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &WAFCollector{
		wafClient: map[string]*wafv2.Client{
			"us-east-1": wafv2.NewFromConfig(cfg),
		},
		cfClient: cloudfront.NewFromConfig(cfg),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to collect regional WAFs")
}

func TestWAFCollector_collectScope_ListWebACLsError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	client := wafv2.NewFromConfig(cfg)
	collector := &WAFCollector{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resources := make([]Resource, 0)
	err := collector.collectScope(ctx, client, nil, "us-east-1", waftypes.ScopeRegional, &resources)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list web ACLs")
	assert.Empty(t, resources)
}
