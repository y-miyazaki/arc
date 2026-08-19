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

			collector, err := NewELBCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.Len(t, collector.elbClients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.elbClients, region)
			}
			assert.Len(t, collector.wafClients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.wafClients, region)
			}
			assert.Len(t, collector.ec2Clients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.ec2Clients, region)
			}
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestELBCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "elb", wantSort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &ELBCollector{
				elbClients: map[string]*elasticloadbalancingv2.Client{},
				wafClients: map[string]*wafv2.Client{},
				ec2Clients: map[string]*ec2.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestELBCollector_GetColumns(t *testing.T) {
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
				Category:     "ELB",
				SubCategory1: "LoadBalancer",
				SubCategory2: "Application",
				Name:         "test-alb",
				Region:       "us-east-1",
				ARN:          "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890123456",
				RawData: map[string]any{
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
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ARN",
				"DNSName", "Type", "VPC", "AvailabilityZone", "SecurityGroup", "WAF",
				"Protocol", "Port", "HealthCheck", "SSLPolicy", "State", "CreatedTime",
			},
			wantValues: []string{
				"ELB", "LoadBalancer", "Application", "test-alb", "us-east-1", "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-alb/1234567890123456",
				"test-alb-123456789.us-east-1.elb.amazonaws.com", "application", "vpc-12345678 (my-vpc)", "us-east-1a, us-east-1b", "sg-12345678 (my-sg)", "WebACL-Test",
				"HTTPS", "443", "/health", "ELBSecurityPolicy-TLS13-1-2-2021-06", "active", "2023-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &ELBCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
