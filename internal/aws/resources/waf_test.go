package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewWAFCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewWAFCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.wafClient, 2)
	assert.Contains(t, collector.wafClient, "us-east-1")
	assert.Contains(t, collector.wafClient, "eu-west-1")
	assert.NotNil(t, collector.cfClient)
	assert.NotNil(t, collector.nameResolver)
}

func TestNewWAFCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewWAFCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.wafClient)
	assert.NotNil(t, collector.cfClient)
	assert.NotNil(t, collector.nameResolver)
}

func TestWAFCollector_Basic(t *testing.T) {
	collector := &WAFCollector{
		wafClient: make(map[string]*wafv2.Client),
	}
	assert.Equal(t, "waf", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestWAFCollector_GetColumns(t *testing.T) {
	collector := &WAFCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region", "ARN",
		"Description", "Scope", "Rules", "AssociatedResources", "Logging",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Security",
		SubCategory1: "WAF",
		Name:         "test-web-acl",
		Region:       "us-east-1",
		ARN:          "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/test-web-acl/12345678-1234-1234-1234-123456789012",
		RawData: map[string]interface{}{
			"Description":         "Test WebACL",
			"Scope":               "REGIONAL",
			"Rules":               []string{"Rule1", "Rule2"},
			"AssociatedResources": []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890abcdef"},
			"Logging":             []string{"arn:aws:s3:::aws-waf-logs-test"},
		},
	}

	expectedValues := []string{
		"Security", "WAF", "test-web-acl", "us-east-1", "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/test-web-acl/12345678-1234-1234-1234-123456789012",
		"Test WebACL", "REGIONAL", "Rule1\nRule2", "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890abcdef", "arn:aws:s3:::aws-waf-logs-test",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
