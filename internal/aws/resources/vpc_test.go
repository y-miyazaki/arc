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
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewVPCCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewVPCCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewVPCCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestVPCCollector_Basic(t *testing.T) {
	collector := &VPCCollector{
		clients: make(map[string]*ec2.Client),
	}
	assert.Equal(t, "vpc", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestVPCCollector_GetColumns(t *testing.T) {
	collector := &VPCCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ID",
		"Description", "CIDR", "PublicIP", "Inbound", "Outbound", "Type",
		"Service", "Subnets", "RouteTables", "SecurityGroups", "Settings", "State",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Network",
		SubCategory1: "VPC",
		SubCategory2: "",
		Name:         "test-vpc",
		Region:       "us-east-1",
		RawData: map[string]interface{}{
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
	}

	expectedValues := []string{
		"Network", "VPC", "", "test-vpc", "us-east-1", "vpc-12345678",
		"Test VPC", "10.0.0.0/16", "1.2.3.4", "0.0.0.0/0:80", "0.0.0.0/0:443", "VPC",
		"EC2", "subnet-123", "rtb-123", "sg-123", "EnableDnsSupport=true", "available",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
