package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewEFSCollector(t *testing.T) {
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

			collector, err := NewEFSCollector(cfg, tt.regions, nameResolver)
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

func TestEFSCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "efs", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &EFSCollector{
				clients: map[string]*efs.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestEFSCollector_GetColumns(t *testing.T) {
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
				Category:     "EFS",
				SubCategory1: "FileSystem",
				Name:         "test-filesystem",
				Region:       "us-east-1",
				ARN:          "fs-12345678", // ID column uses ARN field
				RawData: map[string]any{
					"Type":          "REGIONAL",
					"Performance":   "generalPurpose",
					"Throughput":    "bursting",
					"Encrypted":     "true",
					"KmsKey":        "my-kms-key",
					"Size":          "1073741824",
					"Subnet":        "subnet-12345678 (my-subnet)",
					"IPAddress":     "10.0.1.100",
					"SecurityGroup": "sg-12345678 (my-sg)",
					"Path":          "/mnt/efs",
					"UID":           "1000",
					"GID":           "1000",
					"State":         "available",
					"CreationTime":  "2023-09-25T01:07:55Z",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region",
				"ID", "Type", "Performance", "Throughput", "Encrypted",
				"KmsKey", "Size", "Subnet", "IPAddress", "SecurityGroup",
				"Path", "UID", "GID", "State", "CreationTime",
			},
			wantValues: []string{
				"EFS", "FileSystem", "test-filesystem", "us-east-1",
				"fs-12345678", "REGIONAL", "generalPurpose", "bursting", "true",
				"my-kms-key", "1073741824", "subnet-12345678 (my-subnet)", "10.0.1.100", "sg-12345678 (my-sg)",
				"/mnt/efs", "1000", "1000", "available", "2023-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &EFSCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
