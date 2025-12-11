package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewEC2Collector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewEC2Collector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewEC2Collector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewEC2Collector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestEC2Collector_Basic(t *testing.T) {
	collector := &EC2Collector{
		clients: map[string]*ec2.Client{},
	}
	assert.Equal(t, "ec2", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestEC2Collector_GetColumns(t *testing.T) {
	collector := &EC2Collector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region",
		"InstanceID", "InstanceType", "ImageID", "VPC", "Subnet",
		"SecurityGroup", "State",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "EC2",
		SubCategory1: "Instance",
		Name:         "test-instance",
		Region:       "us-east-1",
		RawData: map[string]interface{}{
			"InstanceID":    "i-1234567890abcdef0",
			"InstanceType":  "t3.micro",
			"ImageID":       "ami-12345678",
			"VPC":           "vpc-12345678 (my-vpc)",
			"Subnet":        "subnet-12345678 (my-subnet)",
			"SecurityGroup": "sg-12345678 (my-sg)",
			"State":         "running",
		},
	}

	expectedValues := []string{
		"EC2", "Instance", "test-instance", "us-east-1",
		"i-1234567890abcdef0", "t3.micro", "ami-12345678", "vpc-12345678 (my-vpc)", "subnet-12345678 (my-subnet)",
		"sg-12345678 (my-sg)", "running",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
