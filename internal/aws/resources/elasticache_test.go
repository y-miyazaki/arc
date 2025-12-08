package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockElastiCacheCollector is a testable version of ElastiCacheCollector that uses mock data
type MockElastiCacheCollector struct{}

func NewMockElastiCacheCollector() *MockElastiCacheCollector {
	return &MockElastiCacheCollector{}
}

func (c *MockElastiCacheCollector) Name() string {
	return "elasticache"
}

func (c *MockElastiCacheCollector) ShouldSort() bool {
	return false
}

func (c *MockElastiCacheCollector) GetColumns() []Column {
	return []Column{
		{Header: "Category", Value: func(r Resource) string { return r.Category }},
		{Header: "SubCategory", Value: func(r Resource) string { return r.SubCategory }},
		{Header: "SubSubCategory", Value: func(r Resource) string { return r.SubSubCategory }},
		{Header: "Name", Value: func(r Resource) string { return r.Name }},
		{Header: "Region", Value: func(r Resource) string { return r.Region }},
		{Header: "ARN", Value: func(r Resource) string { return r.ARN }},
		{Header: "Description", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Description") }},
		{Header: "ReplicationGroupID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ReplicationGroupID") }},
		{Header: "ClusterID", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "ClusterID") }},
		{Header: "Engine", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Engine") }},
		{Header: "Version", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Version") }},
		{Header: "NodeType", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NodeType") }},
		{Header: "NodeGroups", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NodeGroups") }},
		{Header: "NumNodes", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "NumNodes") }},
		{Header: "CacheParameterGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CacheParameterGroup") }},
		{Header: "SecurityGroup", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SecurityGroup") }},
		{Header: "MultiAZ", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "MultiAZ") }},
		{Header: "AutomaticFailover", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AutomaticFailover") }},
		{Header: "EncryptedAtRest", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EncryptedAtRest") }},
		{Header: "EncryptedTransit", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "EncryptedTransit") }},
		{Header: "AuthTokenEnabled", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AuthTokenEnabled") }},
		{Header: "AutoMinorVersionUpgrade", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "AutoMinorVersionUpgrade") }},
		{Header: "PreferredMaintenanceWindow", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "PreferredMaintenanceWindow") }},
		{Header: "SnapshotRetentionLimit", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SnapshotRetentionLimit") }},
		{Header: "SnapshotWindow", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "SnapshotWindow") }},
		{Header: "CreateTime", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "CreateTime") }},
		{Header: "Status", Value: func(r Resource) string { return helpers.GetMapValue(r.RawData, "Status") }},
	}
}

func (c *MockElastiCacheCollector) Collect(ctx context.Context, cfg *aws.Config, region string) ([]Resource, error) {
	// Return mock data without using actual AWS API calls
	var resources []Resource

	// Mock replication group
	r1 := Resource{
		Category:    "elasticache",
		SubCategory: "ReplicationGroup",
		Name:        "my-redis-cluster",
		Region:      region,
		ARN:         "arn:aws:elasticache:us-east-1:123456789012:replicationgroup:my-redis-cluster",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description":                "Redis cluster for caching",
			"ReplicationGroupID":         "my-redis-cluster",
			"ClusterID":                  "N/A",
			"Engine":                     "redis",
			"Version":                    "6.2",
			"NodeType":                   "cache.t3.micro",
			"NodeGroups":                 "1",
			"NumNodes":                   "2",
			"CacheParameterGroup":        "default.redis6.x",
			"SecurityGroup":              "sg-12345,sg-67890",
			"MultiAZ":                    "false",
			"AutomaticFailover":          "enabled",
			"EncryptedAtRest":            "true",
			"EncryptedTransit":           "true",
			"AuthTokenEnabled":           "false",
			"AutoMinorVersionUpgrade":    "true",
			"PreferredMaintenanceWindow": "sun:05:00-sun:06:00",
			"SnapshotRetentionLimit":     "7",
			"SnapshotWindow":             "03:00-04:00",
			"CreateTime":                 "2023-08-15T10:30:00Z",
			"Status":                     "available",
		}),
	}
	resources = append(resources, r1)

	// Mock cache cluster
	r2 := Resource{
		Category:    "elasticache",
		SubCategory: "CacheCluster",
		Name:        "my-memcached-cluster",
		Region:      region,
		ARN:         "arn:aws:elasticache:us-east-1:123456789012:cluster:my-memcached-cluster",
		RawData: helpers.NormalizeRawData(map[string]any{
			"Description":                "Memcached cluster for sessions",
			"ReplicationGroupID":         "N/A",
			"ClusterID":                  "my-memcached-cluster",
			"Engine":                     "memcached",
			"Version":                    "1.6.6",
			"NodeType":                   "cache.t3.small",
			"NodeGroups":                 "N/A",
			"NumNodes":                   "3",
			"CacheParameterGroup":        "default.memcached1.6",
			"SecurityGroup":              "sg-54321",
			"MultiAZ":                    "N/A",
			"AutomaticFailover":          "N/A",
			"EncryptedAtRest":            "false",
			"EncryptedTransit":           "false",
			"AuthTokenEnabled":           "N/A",
			"AutoMinorVersionUpgrade":    "false",
			"PreferredMaintenanceWindow": "mon:04:00-mon:05:00",
			"SnapshotRetentionLimit":     "0",
			"SnapshotWindow":             "N/A",
			"CreateTime":                 "2023-09-01T14:20:00Z",
			"Status":                     "available",
		}),
	}
	resources = append(resources, r2)

	return resources, nil
}

func TestElastiCacheCollector_Basic(t *testing.T) {
	collector := &ElastiCacheCollector{}
	assert.Equal(t, "elasticache", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestElastiCacheCollector_GetColumns(t *testing.T) {
	collector := &ElastiCacheCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory", "SubSubCategory", "Name", "Region",
		"ARN", "Description", "ReplicationGroupID", "ClusterID", "Engine",
		"Version", "NodeType", "NodeGroups", "NumNodes", "CacheParameterGroup",
		"SecurityGroup", "MultiAZ", "AutomaticFailover", "EncryptedAtRest",
		"EncryptedTransit", "AuthTokenEnabled", "AutoMinorVersionUpgrade",
		"PreferredMaintenanceWindow", "SnapshotRetentionLimit", "SnapshotWindow",
		"CreateTime", "Status",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i := range expectedHeaders {
		assert.Equal(t, expectedHeaders[i], columns[i].Header)
	}

	// Test Value functions with a sample resource
	sampleResource := Resource{
		Category:       "Database",
		SubCategory:    "ElastiCache",
		SubSubCategory: "ReplicationGroup",
		Name:           "my-redis-cluster",
		Region:         "us-east-1",
		ARN:            "arn:aws:elasticache:us-east-1:123456789012:replicationgroup:my-redis-cluster",
		RawData: map[string]any{
			"Description":                "Redis cluster for caching",
			"ReplicationGroupID":         "my-redis-cluster",
			"ClusterID":                  "N/A",
			"Engine":                     "redis",
			"Version":                    "6.2",
			"NodeType":                   "cache.t3.micro",
			"NodeGroups":                 "1",
			"NumNodes":                   "2",
			"CacheParameterGroup":        "default.redis6.x",
			"SecurityGroup":              "sg-12345,sg-67890",
			"MultiAZ":                    "false",
			"AutomaticFailover":          "enabled",
			"EncryptedAtRest":            "true",
			"EncryptedTransit":           "true",
			"AuthTokenEnabled":           "false",
			"AutoMinorVersionUpgrade":    "true",
			"PreferredMaintenanceWindow": "sun:05:00-sun:06:00",
			"SnapshotRetentionLimit":     "7",
			"SnapshotWindow":             "03:00-04:00",
			"CreateTime":                 "2023-08-15T10:30:00Z",
			"Status":                     "available",
		},
	}

	expectedValues := []string{
		"Database", "ElastiCache", "ReplicationGroup", "my-redis-cluster", "us-east-1",
		"arn:aws:elasticache:us-east-1:123456789012:replicationgroup:my-redis-cluster",
		"Redis cluster for caching", "my-redis-cluster", "N/A", "redis",
		"6.2", "cache.t3.micro", "1", "2", "default.redis6.x",
		"sg-12345,sg-67890", "false", "enabled", "true",
		"true", "false", "true", "sun:05:00-sun:06:00", "7", "03:00-04:00",
		"2023-08-15T10:30:00Z", "available",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, expectedHeaders[i])
	}
}

func TestMockElastiCacheCollector_Collect(t *testing.T) {
	ctx := context.Background()
	cfg := &aws.Config{}
	region := "us-east-1"

	collector := NewMockElastiCacheCollector()

	resources, err := collector.Collect(ctx, cfg, region)

	assert.NoError(t, err)
	assert.Len(t, resources, 2)

	// Check first resource (Replication Group)
	r1 := resources[0]
	assert.Equal(t, "elasticache", r1.Category)
	assert.Equal(t, "ReplicationGroup", r1.SubCategory)
	assert.Equal(t, "my-redis-cluster", r1.Name)
	assert.Equal(t, region, r1.Region)
	assert.Equal(t, "arn:aws:elasticache:us-east-1:123456789012:replicationgroup:my-redis-cluster", r1.ARN)
	assert.Equal(t, "Redis cluster for caching", helpers.GetMapValue(r1.RawData, "Description"))
	assert.Equal(t, "my-redis-cluster", helpers.GetMapValue(r1.RawData, "ReplicationGroupID"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r1.RawData, "ClusterID"))
	assert.Equal(t, "redis", helpers.GetMapValue(r1.RawData, "Engine"))
	assert.Equal(t, "6.2", helpers.GetMapValue(r1.RawData, "Version"))
	assert.Equal(t, "cache.t3.micro", helpers.GetMapValue(r1.RawData, "NodeType"))
	assert.Equal(t, "1", helpers.GetMapValue(r1.RawData, "NodeGroups"))
	assert.Equal(t, "2", helpers.GetMapValue(r1.RawData, "NumNodes"))
	assert.Equal(t, "default.redis6.x", helpers.GetMapValue(r1.RawData, "CacheParameterGroup"))
	assert.Equal(t, "sg-12345,sg-67890", helpers.GetMapValue(r1.RawData, "SecurityGroup"))
	assert.Equal(t, "false", helpers.GetMapValue(r1.RawData, "MultiAZ"))
	assert.Equal(t, "enabled", helpers.GetMapValue(r1.RawData, "AutomaticFailover"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "EncryptedAtRest"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "EncryptedTransit"))
	assert.Equal(t, "false", helpers.GetMapValue(r1.RawData, "AuthTokenEnabled"))
	assert.Equal(t, "true", helpers.GetMapValue(r1.RawData, "AutoMinorVersionUpgrade"))
	assert.Equal(t, "sun:05:00-sun:06:00", helpers.GetMapValue(r1.RawData, "PreferredMaintenanceWindow"))
	assert.Equal(t, "7", helpers.GetMapValue(r1.RawData, "SnapshotRetentionLimit"))
	assert.Equal(t, "03:00-04:00", helpers.GetMapValue(r1.RawData, "SnapshotWindow"))
	assert.Equal(t, "2023-08-15T10:30:00Z", helpers.GetMapValue(r1.RawData, "CreateTime"))
	assert.Equal(t, "available", helpers.GetMapValue(r1.RawData, "Status"))

	// Check second resource (Cache Cluster)
	r2 := resources[1]
	assert.Equal(t, "elasticache", r2.Category)
	assert.Equal(t, "CacheCluster", r2.SubCategory)
	assert.Equal(t, "my-memcached-cluster", r2.Name)
	assert.Equal(t, region, r2.Region)
	assert.Equal(t, "arn:aws:elasticache:us-east-1:123456789012:cluster:my-memcached-cluster", r2.ARN)
	assert.Equal(t, "Memcached cluster for sessions", helpers.GetMapValue(r2.RawData, "Description"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "ReplicationGroupID"))
	assert.Equal(t, "my-memcached-cluster", helpers.GetMapValue(r2.RawData, "ClusterID"))
	assert.Equal(t, "memcached", helpers.GetMapValue(r2.RawData, "Engine"))
	assert.Equal(t, "1.6.6", helpers.GetMapValue(r2.RawData, "Version"))
	assert.Equal(t, "cache.t3.small", helpers.GetMapValue(r2.RawData, "NodeType"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "NodeGroups"))
	assert.Equal(t, "3", helpers.GetMapValue(r2.RawData, "NumNodes"))
	assert.Equal(t, "default.memcached1.6", helpers.GetMapValue(r2.RawData, "CacheParameterGroup"))
	assert.Equal(t, "sg-54321", helpers.GetMapValue(r2.RawData, "SecurityGroup"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "MultiAZ"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "AutomaticFailover"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "EncryptedAtRest"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "EncryptedTransit"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "AuthTokenEnabled"))
	assert.Equal(t, "false", helpers.GetMapValue(r2.RawData, "AutoMinorVersionUpgrade"))
	assert.Equal(t, "mon:04:00-mon:05:00", helpers.GetMapValue(r2.RawData, "PreferredMaintenanceWindow"))
	assert.Equal(t, "0", helpers.GetMapValue(r2.RawData, "SnapshotRetentionLimit"))
	assert.Equal(t, "N/A", helpers.GetMapValue(r2.RawData, "SnapshotWindow"))
	assert.Equal(t, "2023-09-01T14:20:00Z", helpers.GetMapValue(r2.RawData, "CreateTime"))
	assert.Equal(t, "available", helpers.GetMapValue(r2.RawData, "Status"))
}
