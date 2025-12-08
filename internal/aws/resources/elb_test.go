package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewELBCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewELBCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.elbClients)
	assert.Len(t, collector.elbClients, len(regions))
	assert.NotNil(t, collector.wafClients)
	assert.Len(t, collector.wafClients, len(regions))
	assert.NotNil(t, collector.ec2Clients)
	assert.Len(t, collector.ec2Clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewELBCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewELBCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.elbClients)
	assert.Len(t, collector.elbClients, 0)
	assert.NotNil(t, collector.wafClients)
	assert.Len(t, collector.wafClients, 0)
	assert.NotNil(t, collector.ec2Clients)
	assert.Len(t, collector.ec2Clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestELBCollector_Basic(t *testing.T) {
	collector := &ELBCollector{
		elbClients: map[string]*elasticloadbalancingv2.Client{},
		wafClients: map[string]*wafv2.Client{},
		ec2Clients: map[string]*ec2.Client{},
	}
	assert.Equal(t, "elb", collector.Name())
	assert.False(t, collector.ShouldSort()) // ELB should not be sorted
}

func TestELBCollector_GetColumns(t *testing.T) {
	collector := &ELBCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"DNSName", "Type", "VPC", "AvailabilityZone", "SecurityGroup", "WAF",
		"Protocol", "Port", "HealthCheck", "SSLPolicy", "State", "CreatedTime",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "ELB",
		SubCategory:    "LoadBalancer",
		SubSubCategory: "Application",
		Name:           "test-alb",
		Region:         "us-east-1",
		ARN:            "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890123456",
		RawData: map[string]interface{}{
			"DNSName":          "test-alb-123456789.us-east-1.elb.amazonaws.com",
			"Type":             "application",
			"VPC":              "vpc-12345678 (my-vpc)",
			"AvailabilityZone": "us-east-1a, us-east-1b",
			"SecurityGroup":    "sg-12345678 (my-sg)",
			"WAF":              "WebACL-Test",
			"Protocol":         "HTTPS",
			"Port":             "443",
			"HealthCheck":      "/health",
			"SSLPolicy":        "ELBSecurityPolicy-TLS13-1-2-2021-06",
			"State":            "active",
			"CreatedTime":      "2023-09-25T01:07:55Z",
		},
	}

	expectedValues := []string{
		"ELB", "LoadBalancer", "Application", "test-alb", "us-east-1", "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890123456",
		"test-alb-123456789.us-east-1.elb.amazonaws.com", "application", "vpc-12345678 (my-vpc)", "us-east-1a, us-east-1b", "sg-12345678 (my-sg)", "WebACL-Test",
		"HTTPS", "443", "/health", "ELBSecurityPolicy-TLS13-1-2-2021-06", "active", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
