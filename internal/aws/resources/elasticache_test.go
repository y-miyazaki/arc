package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewElastiCacheCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewElastiCacheCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewElastiCacheCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewElastiCacheCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestElastiCacheCollector_Basic(t *testing.T) {
	collector := &ElastiCacheCollector{
		clients: map[string]*elasticache.Client{},
	}
	assert.Equal(t, "elasticache", collector.Name())
	assert.False(t, collector.ShouldSort()) // ElastiCache should not be sorted
}

func TestElastiCacheCollector_GetColumns(t *testing.T) {
	collector := &ElastiCacheCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region", "ARN",
		"Description", "ReplicationGroupID", "ClusterID", "Engine", "Version",
		"NodeType", "NodeGroups", "NumNodes", "CacheParameterGroup", "SecurityGroup",
		"MultiAZ", "AutomaticFailover", "EncryptedAtRest", "EncryptedTransit",
		"AuthTokenEnabled", "AutoMinorVersionUpgrade", "PreferredMaintenanceWindow",
		"SnapshotRetentionLimit", "SnapshotWindow", "CreateTime", "Status",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:       "ElastiCache",
		SubCategory:    "ReplicationGroup",
		SubSubCategory: "",
		Name:           "test-cluster",
		Region:         "us-east-1",
		ARN:            "arn:aws:elasticache:us-east-1:123456789012:replicationgroup:test-cluster",
		RawData: map[string]interface{}{
			"Description":                "Test cluster description",
			"ReplicationGroupID":         "test-replication-group",
			"ClusterID":                  "test-cluster-001",
			"Engine":                     "redis",
			"Version":                    "6.2.6",
			"NodeType":                   "cache.t3.micro",
			"NodeGroups":                 "1",
			"NumNodes":                   "2",
			"CacheParameterGroup":        "default.redis6.x",
			"SecurityGroup":              "sg-12345678 (my-sg)",
			"MultiAZ":                    "enabled",
			"AutomaticFailover":          "enabled",
			"EncryptedAtRest":            "true",
			"EncryptedTransit":           "true",
			"AuthTokenEnabled":           "true",
			"AutoMinorVersionUpgrade":    "true",
			"PreferredMaintenanceWindow": "sun:05:00-sun:06:00",
			"SnapshotRetentionLimit":     "7",
			"SnapshotWindow":             "03:00-04:00",
			"CreateTime":                 "2023-09-25T01:07:55Z",
			"Status":                     "available",
		},
	}

	expectedValues := []string{
		"ElastiCache", "ReplicationGroup", "", "test-cluster", "us-east-1", "arn:aws:elasticache:us-east-1:123456789012:replicationgroup:test-cluster",
		"Test cluster description", "test-replication-group", "test-cluster-001", "redis", "6.2.6",
		"cache.t3.micro", "1", "2", "default.redis6.x", "sg-12345678 (my-sg)",
		"enabled", "enabled", "true", "true",
		"true", "true", "sun:05:00-sun:06:00",
		"7", "03:00-04:00", "2023-09-25T01:07:55Z", "available",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
