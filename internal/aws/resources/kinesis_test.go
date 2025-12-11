package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewKinesisCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewKinesisCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.kinesisClients)
	assert.Len(t, collector.kinesisClients, len(regions))
	assert.NotNil(t, collector.firehoseClients)
	assert.Len(t, collector.firehoseClients, len(regions))
	assert.NotNil(t, collector.nameResolver)
}

func TestNewKinesisCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewKinesisCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.kinesisClients)
	assert.Len(t, collector.kinesisClients, 0)
	assert.NotNil(t, collector.firehoseClients)
	assert.Len(t, collector.firehoseClients, 0)
	assert.NotNil(t, collector.nameResolver)
}

func TestKinesisCollector_Basic(t *testing.T) {
	collector := &KinesisCollector{
		kinesisClients:  map[string]*kinesis.Client{},
		firehoseClients: map[string]*firehose.Client{},
	}
	assert.Equal(t, "kinesis", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestKinesisCollector_GetColumns(t *testing.T) {
	collector := &KinesisCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region",
		"ARN", "Status", "Shards", "DestinationId", "RetentionPeriodHours",
		"EncryptionType", "CreatedDate", "LastUpdatedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Analytics",
		SubCategory1: "Kinesis",
		Name:         "test-stream",
		Region:       "us-east-1",
		ARN:          "arn:aws:kinesis:us-east-1:123456789012:stream/test-stream",
		RawData: map[string]interface{}{
			"Status":               "ACTIVE",
			"Shards":               "2",
			"DestinationId":        "",
			"RetentionPeriodHours": "24",
			"EncryptionType":       "KMS",
			"CreatedDate":          "2023-09-25T01:07:55Z",
			"LastUpdatedDate":      "2023-09-26T10:30:00Z",
		},
	}

	expectedValues := []string{
		"Analytics", "Kinesis", "test-stream", "us-east-1",
		"arn:aws:kinesis:us-east-1:123456789012:stream/test-stream", "ACTIVE", "2", "", "24",
		"KMS", "2023-09-25T01:07:55Z", "2023-09-26T10:30:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
