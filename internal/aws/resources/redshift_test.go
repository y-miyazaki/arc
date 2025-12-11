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
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewRedshiftCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewRedshiftCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewRedshiftCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.clients)
	assert.Len(t, collector.clients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestRedshiftCollector_Basic(t *testing.T) {
	collector := &RedshiftCollector{
		clients: map[string]*redshift.Client{},
	}
	assert.Equal(t, "redshift", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestRedshiftCollector_GetColumns(t *testing.T) {
	collector := &RedshiftCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region",
		"RoleARN", "NodeType", "NumberOfNodes", "DBName", "Endpoint",
		"Port", "MasterUsername", "VPCName", "ClusterSubnetGroupName", "SecurityGroup",
		"Encrypted", "KmsKey", "PubliclyAccessible", "ClusterStatus",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Database",
		SubCategory1: "Redshift",
		SubCategory2: "Cluster",
		Name:         "test-cluster",
		Region:       "us-east-1",
		ARN:          "arn:aws:iam::123456789012:role/RedshiftRole",
		RawData: map[string]interface{}{
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
	}

	expectedValues := []string{
		"Database", "Redshift", "test-cluster", "us-east-1",
		"arn:aws:iam::123456789012:role/RedshiftRole", "dc2.large", "2", "mydb", "test-cluster.cluster-random.us-east-1.redshift.amazonaws.com",
		"5439", "admin", "vpc-prod", "redshift-subnet-group", "sg-12345678",
		"true", "alias/aws/redshift", "false", "available",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
