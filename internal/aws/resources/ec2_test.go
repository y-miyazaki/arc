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

			collector, err := NewEC2Collector(cfg, tt.regions, nameResolver)
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

func TestEC2Collector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "ec2", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &EC2Collector{
				clients: map[string]*ec2.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestEC2Collector_GetColumns(t *testing.T) {
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
				Category:     "EC2",
				SubCategory1: "Instance",
				Name:         "test-instance",
				Region:       "us-east-1",
				RawData: map[string]any{
					"InstanceID":    "i-1234567890abcdef0",
					"InstanceType":  "t3.micro",
					"ImageID":       "ami-12345678",
					"VPC":           "vpc-12345678 (my-vpc)",
					"Subnet":        "subnet-12345678 (my-subnet)",
					"SecurityGroup": "sg-12345678 (my-sg)",
					"State":         "running",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region",
				"InstanceID", "InstanceType", "ImageID", "VPC", "Subnet",
				"SecurityGroup", "State",
			},
			wantValues: []string{
				"EC2", "Instance", "test-instance", "us-east-1",
				"i-1234567890abcdef0", "t3.micro", "ami-12345678", "vpc-12345678 (my-vpc)", "subnet-12345678 (my-subnet)",
				"sg-12345678 (my-sg)", "running",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &EC2Collector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
