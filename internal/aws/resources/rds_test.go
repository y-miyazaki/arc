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

			collector, err := NewRDSCollector(cfg, tt.regions, nameResolver)
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

func TestRDSCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "rds", wantSort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &RDSCollector{
				clients: map[string]*rds.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestRDSCollector_GetColumns(t *testing.T) {
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
				Category:     "Database",
				SubCategory1: "RDS",
				SubCategory2: "DBInstance",
				Name:         "test-db",
				Region:       "us-east-1",
				RawData: map[string]any{
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
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "SubCategory2", "Name", "Region",
				"ID", "Type", "Engine", "Version", "InstanceClass",
				"AllocatedStorage", "MultiAZ", "DBClusterMembers", "EngineLifecycleSupport", "IAMDatabaseAuthenticationEnabled",
				"KerberosAuth", "KmsKey", "AvailabilityZone", "BackupRetentionPeriod",
			},
			wantValues: []string{
				"Database", "RDS", "DBInstance", "test-db", "us-east-1",
				"test-db", "DBInstance", "mysql", "8.0.32", "db.t3.micro",
				"20", "false", "0", "open-source-rds-extended-support", "false",
				"false", "alias/aws/rds", "us-east-1a", "7",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &RDSCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
