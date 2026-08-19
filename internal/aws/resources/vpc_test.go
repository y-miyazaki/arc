package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewVPCCollector(t *testing.T) {
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

			collector, err := NewVPCCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.Len(t, collector.clients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.clients, region)
			}
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestVPCCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "vpc", wantSort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &VPCCollector{
				clients: make(map[string]*ec2.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestVPCCollector_GetColumns(t *testing.T) {
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
				Category:     "Network",
				SubCategory1: "VPC",
				SubCategory2: "",
				Name:         "test-vpc",
				Region:       "us-east-1",
				RawData: map[string]any{
					"ID":             "vpc-12345678",
					"Description":    "Test VPC",
					"CIDR":           "10.0.0.0/16",
					"PublicIP":       "1.2.3.4",
					"Inbound":        "0.0.0.0/0:80",
					"Outbound":       "0.0.0.0/0:443",
					"Type":           "VPC",
					"Service":        "EC2",
					"Subnets":        "subnet-123",
					"RouteTables":    "rtb-123",
					"SecurityGroups": "sg-123",
					"Settings":       "EnableDnsSupport=true",
					"State":          "available",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ID",
				"Description", "CIDR", "PublicIP", "Inbound", "Outbound", "Type",
				"Service", "Subnets", "RouteTables", "SecurityGroups", "Settings", "State",
			},
			wantValues: []string{
				"Network", "VPC", "", "test-vpc", "us-east-1", "vpc-12345678",
				"Test VPC", "10.0.0.0/16", "1.2.3.4", "0.0.0.0/0:80", "0.0.0.0/0:443", "VPC",
				"EC2", "subnet-123", "rtb-123", "sg-123", "EnableDnsSupport=true", "available",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &VPCCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
