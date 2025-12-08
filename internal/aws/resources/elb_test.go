package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestELBCollector_Name(t *testing.T) {
	collector := &ELBCollector{}
	assert.Equal(t, "elb", collector.Name())
}

func TestELBCollector_ShouldSort(t *testing.T) {
	collector := &ELBCollector{}
	assert.False(t, collector.ShouldSort())
}

func TestELBCollector_GetColumns(t *testing.T) {
	collector := &ELBCollector{}
	columns := collector.GetColumns()

	assert.NotEmpty(t, columns)
	assert.Contains(t, columns[0].Header, "Category")
	assert.Contains(t, columns[1].Header, "SubCategory")
	assert.Contains(t, columns[2].Header, "SubSubCategory")
	assert.Contains(t, columns[3].Header, "Name")
	assert.Contains(t, columns[4].Header, "Region")
	assert.Contains(t, columns[5].Header, "ARN")
	assert.Contains(t, columns[6].Header, "DNSName")

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "elb",
		SubCategory:    "LoadBalancer",
		SubSubCategory: "Application",
		Name:           "test-alb",
		Region:         "us-east-1",
		ARN:            "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890123456",
		RawData: map[string]any{
			"DNSName":          "test-alb-1234567890.us-east-1.elb.amazonaws.com",
			"Type":             "application",
			"VPC":              "vpc-12345678",
			"AvailabilityZone": "us-east-1a",
			"SecurityGroup":    "sg-12345678",
			"WAF":              "waf-12345678",
			"Protocol":         "HTTP",
			"Port":             "80",
			"HealthCheck":      "HTTP:80/health",
		},
	}

	// Test each Value function
	assert.Equal(t, "elb", columns[0].Value(sampleResource))
	assert.Equal(t, "LoadBalancer", columns[1].Value(sampleResource))
	assert.Equal(t, "Application", columns[2].Value(sampleResource))
	assert.Equal(t, "test-alb", columns[3].Value(sampleResource))
	assert.Equal(t, "us-east-1", columns[4].Value(sampleResource))
	assert.Equal(t, "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890123456", columns[5].Value(sampleResource))
	assert.Equal(t, "test-alb-1234567890.us-east-1.elb.amazonaws.com", columns[6].Value(sampleResource))
	assert.Equal(t, "application", columns[7].Value(sampleResource))
	assert.Equal(t, "vpc-12345678", columns[8].Value(sampleResource))
	assert.Equal(t, "us-east-1a", columns[9].Value(sampleResource))
	assert.Equal(t, "sg-12345678", columns[10].Value(sampleResource))
	assert.Equal(t, "waf-12345678", columns[11].Value(sampleResource))
	assert.Equal(t, "HTTP", columns[12].Value(sampleResource))
	assert.Equal(t, "80", columns[13].Value(sampleResource))
	assert.Equal(t, "HTTP:80/health", columns[14].Value(sampleResource))
}

// MockELBCollector is a mock implementation of ELBCollector for testing
type MockELBCollector struct{}

func (m *MockELBCollector) Name() string {
	return "elb"
}

func (m *MockELBCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	return []Resource{
		{
			Category:    "elb",
			SubCategory: "LoadBalancer",
			Name:        "test-alb",
			Region:      region,
			ARN:         "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890123456",
			RawData: map[string]any{
				"DNSName":          "test-alb-1234567890.us-east-1.elb.amazonaws.com",
				"Type":             "application",
				"VPC":              "vpc-12345",
				"AvailabilityZone": "us-east-1a,us-east-1b",
				"SecurityGroup":    "sg-12345",
				"WAF":              "waf-id",
			},
		},
		{
			Category:    "elb",
			SubCategory: "TargetGroup",
			Name:        "test-target-group",
			Region:      region,
			ARN:         "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-target-group/1234567890123456",
			RawData: map[string]any{
				"Protocol":    "HTTP",
				"Port":        "80",
				"HealthCheck": "HTTP:80/health",
				"VPC":         "vpc-12345",
			},
		},
	}, nil
}

func (m *MockELBCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "DNSName", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "DNSName") }},
	}
}

func (m *MockELBCollector) ShouldSort() bool {
	return false
}

func TestMockELBCollector_Collect(t *testing.T) {
	collector := &MockELBCollector{}
	cfg := &aws.Config{}
	region := "us-east-1"

	resources, err := collector.Collect(context.Background(), cfg, region)

	assert.NoError(t, err)
	assert.NotEmpty(t, resources)
	assert.Equal(t, 2, len(resources))

	// Check load balancer resource
	lbResource := resources[0]
	assert.Equal(t, "elb", lbResource.Category)
	assert.Equal(t, "LoadBalancer", lbResource.SubCategory)
	assert.Equal(t, "test-alb", lbResource.Name)
	assert.Equal(t, region, lbResource.Region)
	assert.Contains(t, lbResource.ARN, "test-alb")

	// Check target group resource
	tgResource := resources[1]
	assert.Equal(t, "elb", tgResource.Category)
	assert.Equal(t, "TargetGroup", tgResource.SubCategory)
	assert.Equal(t, "test-target-group", tgResource.Name)
	assert.Equal(t, region, tgResource.Region)
	assert.Contains(t, tgResource.ARN, "test-target-group")
}
