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

			collector, err := NewElastiCacheCollector(cfg, tt.regions, nameResolver)
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

func TestElastiCacheCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "elasticache", wantSort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &ElastiCacheCollector{
				clients: map[string]*elasticache.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestElastiCacheCollector_GetColumns(t *testing.T) {
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
				Category:     "ElastiCache",
				SubCategory1: "ReplicationGroup",
				SubCategory2: "",
				Name:         "test-cluster",
				Region:       "us-east-1",
				ARN:          "arn:aws:elasticache:us-east-1:123456789012:replicationgroup:test-cluster",
				RawData: map[string]any{
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
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ARN",
				"Description", "ReplicationGroupID", "ClusterID", "Engine", "Version",
				"NodeType", "NodeGroups", "NumNodes", "CacheParameterGroup", "SecurityGroup",
				"MultiAZ", "AutomaticFailover", "EncryptedAtRest", "EncryptedTransit",
				"AuthTokenEnabled", "AutoMinorVersionUpgrade", "PreferredMaintenanceWindow",
				"SnapshotRetentionLimit", "SnapshotWindow", "Status", "CreateTime",
			},
			wantValues: []string{
				"ElastiCache", "ReplicationGroup", "", "test-cluster", "us-east-1", "arn:aws:elasticache:us-east-1:123456789012:replicationgroup:test-cluster",
				"Test cluster description", "test-replication-group", "test-cluster-001", "redis", "6.2.6",
				"cache.t3.micro", "1", "2", "default.redis6.x", "sg-12345678 (my-sg)",
				"enabled", "enabled", "true", "true",
				"true", "true", "sun:05:00-sun:06:00",
				"7", "03:00-04:00", "available", "2023-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &ElastiCacheCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
