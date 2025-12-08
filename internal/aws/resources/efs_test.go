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
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewEFSCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewEFSCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewEFSCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestEFSCollector_Basic(t *testing.T) {
	collector := &EFSCollector{
		clients: map[string]*efs.Client{},
	}
	assert.Equal(t, "efs", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestEFSCollector_GetColumns(t *testing.T) {
	collector := &EFSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "Type", "Performance", "Throughput", "Encrypted",
		"KmsKey", "Size", "Subnet", "IPAddress", "SecurityGroup",
		"Path", "UID", "GID", "State", "CreationTime",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "EFS",
		SubCategory:    "FileSystem",
		SubSubCategory: "",
		Name:           "test-filesystem",
		Region:         "us-east-1",
		ARN:            "fs-12345678", // ID column uses ARN field
		RawData: map[string]interface{}{
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
	}

	expectedValues := []string{
		"EFS", "FileSystem", "", "test-filesystem", "us-east-1",
		"fs-12345678", "REGIONAL", "generalPurpose", "bursting", "true",
		"my-kms-key", "1073741824", "subnet-12345678 (my-subnet)", "10.0.1.100", "sg-12345678 (my-sg)",
		"/mnt/efs", "1000", "1000", "available", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
