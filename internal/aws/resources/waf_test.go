package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockWAFCollector is a testable version of WAFCollector that uses mock data
type MockWAFCollector struct{}

func NewMockWAFCollector() *MockWAFCollector {
	return &MockWAFCollector{}
}

func (c *MockWAFCollector) Name() string {
	return "waf"
}

func (c *MockWAFCollector) ShouldSort() bool {
	return true
}

func (c *MockWAFCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "Scope", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Scope") }},
		{Header: "Rules", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Rules") }},
		{Header: "AssociatedResources", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AssociatedResources") }},
		{Header: "Logging", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Logging") }},
	}
}

func (c *MockWAFCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock Regional WebACL
	r1 := Resource{
		Category:    "waf",
		SubCategory: "WebACL",
		Name:        "MyRegionalWebACL",
		Region:      region,
		ARN:         "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/MyRegionalWebACL/12345678-1234-1234-1234-123456789012",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description":         "Regional WAF for ALB protection",
			"Scope":               "REGIONAL",
			"Rules":               []string{"AWSManagedRulesCommonRuleSet", "AWSManagedRulesKnownBadInputsRuleSet"},
			"AssociatedResources": []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/1234567890123456"},
			"Logging":             []string{"arn:aws:logs:us-east-1:123456789012:log-group:aws-waf-logs-myregionalwebacl"},
		}),
	}
	resources = append(resources, r1)

	// Mock CloudFront WebACL (only for us-east-1)
	if region == "us-east-1" {
		r2 := Resource{
			Category:    "waf",
			SubCategory: "WebACL",
			Name:        "MyGlobalWebACL",
			Region:      "Global",
			ARN:         "arn:aws:wafv2::123456789012:global/webacl/MyGlobalWebACL/87654321-4321-4321-4321-210987654321",
			RawData: helpers.NormalizeRawData(map[string]any{
				"Description":         "Global WAF for CloudFront protection",
				"Scope":               "CLOUDFRONT",
				"Rules":               []string{"AWSManagedRulesAmazonIpReputationList"},
				"AssociatedResources": []string{"arn:aws:cloudfront::123456789012:distribution/E1A2B3C4D5F6G7H8"},
				"Logging":             []string{"arn:aws:logs:us-east-1:123456789012:log-group:aws-waf-logs-myglobalwebacl"},
			}),
		}
		resources = append(resources, r2)
	}

	return resources, nil
}

func TestWAFCollector_Basic(t *testing.T) {
	collector := &WAFCollector{}
	assert.Equal(t, "waf", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestWAFCollector_GetColumns(t *testing.T) {
	collector := &WAFCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Description", "Scope", "Rules", "AssociatedResources", "Logging",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Security",
		SubCategory:    "WAF",
		SubSubCategory: "WebACL",
		Name:           "my-web-acl",
		Region:         "us-east-1",
		ARN:            "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/my-web-acl/12345678-1234-1234-1234-123456789012",
		RawData: map[string]interface{}{
			"Description":         "Web ACL for API protection",
			"Scope":               "REGIONAL",
			"Rules":               "3 rules",
			"AssociatedResources": "2 ALBs",
			"Logging":             "Enabled",
		},
	}

	expectedValues := []string{
		"Security", "WAF", "WebACL", "my-web-acl", "us-east-1",
		"arn:aws:wafv2:us-east-1:123456789012:regional/webacl/my-web-acl/12345678-1234-1234-1234-123456789012",
		"Web ACL for API protection", "REGIONAL", "3 rules", "2 ALBs", "Enabled",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}

func TestMockWAFCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockWAFCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (Regional WebACL)
	r1 := resources[0]
	assert.Equal(t, "waf", r1.Category)
	assert.Equal(t, "WebACL", r1.SubCategory)
	assert.Equal(t, "MyRegionalWebACL", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/MyRegionalWebACL/12345678-1234-1234-1234-123456789012", r1.ARN)
	assert.Equal(t, "Regional WAF for ALB protection", helpers.GetMapValue(r1.RawData, "Description"))
	assert.Equal(t, "REGIONAL", helpers.GetMapValue(r1.RawData, "Scope"))
	assert.Equal(t, "AWSManagedRulesCommonRuleSet\nAWSManagedRulesKnownBadInputsRuleSet", helpers.GetMapValue(r1.RawData, "Rules"))
	assert.Equal(t, "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/1234567890123456", helpers.GetMapValue(r1.RawData, "AssociatedResources"))
	assert.Equal(t, "arn:aws:logs:us-east-1:123456789012:log-group:aws-waf-logs-myregionalwebacl", helpers.GetMapValue(r1.RawData, "Logging"))

	// Check second resource (Global WebACL)
	r2 := resources[1]
	assert.Equal(t, "waf", r2.Category)
	assert.Equal(t, "WebACL", r2.SubCategory)
	assert.Equal(t, "MyGlobalWebACL", r2.Name)
	assert.Equal(t, "Global", r2.Region)
	assert.Equal(t, "arn:aws:wafv2::123456789012:global/webacl/MyGlobalWebACL/87654321-4321-4321-4321-210987654321", r2.ARN)
	assert.Equal(t, "Global WAF for CloudFront protection", helpers.GetMapValue(r2.RawData, "Description"))
	assert.Equal(t, "CLOUDFRONT", helpers.GetMapValue(r2.RawData, "Scope"))
	assert.Equal(t, "AWSManagedRulesAmazonIpReputationList", helpers.GetMapValue(r2.RawData, "Rules"))
	assert.Equal(t, "arn:aws:cloudfront::123456789012:distribution/E1A2B3C4D5F6G7H8", helpers.GetMapValue(r2.RawData, "AssociatedResources"))
	assert.Equal(t, "arn:aws:logs:us-east-1:123456789012:log-group:aws-waf-logs-myglobalwebacl", helpers.GetMapValue(r2.RawData, "Logging"))
}

func TestMockWAFCollector_Collect_NonUSEast1(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "eu-west-1"

	collector := NewMockWAFCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 1) // Only regional WebACL, no global one

	// Check resource (Regional WebACL)
	r1 := resources[0]
	assert.Equal(t, "waf", r1.Category)
	assert.Equal(t, "WebACL", r1.SubCategory)
	assert.Equal(t, "MyRegionalWebACL", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "REGIONAL", helpers.GetMapValue(r1.RawData, "Scope"))
}
