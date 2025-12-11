package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewTransferFamilyCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewTransferFamilyCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewTransferFamilyCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewTransferFamilyCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestTransferFamilyCollector_Basic(t *testing.T) {
	collector := &TransferFamilyCollector{
		clients: make(map[string]*transfer.Client),
	}
	assert.Equal(t, "transferfamily", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestTransferFamilyCollector_GetColumns(t *testing.T) {
	collector := &TransferFamilyCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region", "ServerID",
		"Protocol", "State",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "transferfamily",
		SubCategory1: "Server",
		Name:         "s-1234567890abcdef0",
		Region:       "us-east-1",
		ARN:          "s-1234567890abcdef0",
		RawData: map[string]interface{}{
			"Protocol": "SFTP",
			"State":    "ONLINE",
		},
	}

	expectedValues := []string{
		"transferfamily", "Server", "s-1234567890abcdef0", "us-east-1", "s-1234567890abcdef0",
		"SFTP", "ONLINE",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
