package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewRedshiftCollector(t *testing.T) {
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

			collector, err := NewRedshiftCollector(cfg, tt.regions, nameResolver)
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

func TestRedshiftCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "redshift", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &RedshiftCollector{
				clients: map[string]*redshift.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestRedshiftCollector_GetColumns(t *testing.T) {
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
				SubCategory1: "Redshift",
				SubCategory2: "Cluster",
				Name:         "test-cluster",
				Region:       "us-east-1",
				ARN:          "arn:aws:iam::123456789012:role/RedshiftRole",
				RawData: map[string]any{
					"NodeType":               "dc2.large",
					"NumberOfNodes":          "2",
					"DBName":                 "mydb",
					"Endpoint":               "test-cluster.cluster-random.us-east-1.redshift.amazonaws.com",
					"Port":                   "5439",
					"MasterUsername":         "admin",
					"VPCName":                "vpc-prod",
					"ClusterSubnetGroupName": "redshift-subnet-group",
					"SecurityGroup":          "sg-12345678",
					"Encrypted":              "true",
					"KmsKey":                 "alias/aws/redshift",
					"PubliclyAccessible":     "false",
					"ClusterStatus":          "available",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region",
				"RoleARN", "NodeType", "NumberOfNodes", "DBName", "Endpoint",
				"Port", "MasterUsername", "VPCName", "ClusterSubnetGroupName", "SecurityGroup",
				"Encrypted", "KmsKey", "PubliclyAccessible", "ClusterStatus",
			},
			wantValues: []string{
				"Database", "Redshift", "test-cluster", "us-east-1",
				"arn:aws:iam::123456789012:role/RedshiftRole", "dc2.large", "2", "mydb", "test-cluster.cluster-random.us-east-1.redshift.amazonaws.com",
				"5439", "admin", "vpc-prod", "redshift-subnet-group", "sg-12345678",
				"true", "alias/aws/redshift", "false", "available",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &RedshiftCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
