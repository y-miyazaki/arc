package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewRDSCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewRDSCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewRDSCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewRDSCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestRDSCollector_Basic(t *testing.T) {
	collector := &RDSCollector{
		clients: map[string]*rds.Client{},
	}
	assert.Equal(t, "rds", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestRDSCollector_GetColumns(t *testing.T) {
	collector := &RDSCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ID", "Type", "Engine", "Version", "InstanceClass",
		"AllocatedStorage", "MultiAZ", "DBClusterMembers", "EngineLifecycleSupport", "IAMDatabaseAuthenticationEnabled",
		"KerberosAuth", "KmsKey", "AvailabilityZone", "BackupRetentionPeriod",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "Database",
		SubCategory:    "RDS",
		SubSubCategory: "DBInstance",
		Name:           "test-db",
		Region:         "us-east-1",
		RawData: map[string]interface{}{
			"ID":                               "test-db",
			"Type":                             "DBInstance",
			"Engine":                           "mysql",
			"Version":                          "8.0.32",
			"InstanceClass":                    "db.t3.micro",
			"AllocatedStorage":                 "20",
			"MultiAZ":                          "false",
			"DBClusterMembers":                 "0",
			"EngineLifecycleSupport":           "open-source-rds-extended-support",
			"IAMDatabaseAuthenticationEnabled": "false",
			"KerberosAuth":                     "false",
			"KmsKey":                           "alias/aws/rds",
			"AvailabilityZone":                 "us-east-1a",
			"BackupRetentionPeriod":            "7",
		},
	}

	expectedValues := []string{
		"Database", "RDS", "DBInstance", "test-db", "us-east-1",
		"test-db", "DBInstance", "mysql", "8.0.32", "db.t3.micro",
		"20", "false", "0", "open-source-rds-extended-support", "false",
		"false", "alias/aws/rds", "us-east-1a", "7",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
