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

			collector, err := NewKinesisCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.Len(t, collector.kinesisClients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.kinesisClients, region)
			}
			assert.Len(t, collector.firehoseClients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.firehoseClients, region)
			}
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestKinesisCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "kinesis", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &KinesisCollector{
				kinesisClients:  map[string]*kinesis.Client{},
				firehoseClients: map[string]*firehose.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestKinesisCollector_GetColumns(t *testing.T) {
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
				Category:     "Analytics",
				SubCategory1: "Kinesis",
				Name:         "test-stream",
				Region:       "us-east-1",
				ARN:          "arn:aws:kinesis:us-east-1:123456789012:stream/test-stream",
				RawData: map[string]any{
					"Status":               "ACTIVE",
					"Shards":               "2",
					"DestinationId":        "",
					"RetentionPeriodHours": "24",
					"EncryptionType":       "KMS",
					"CreatedDate":          "2023-09-25T01:07:55Z",
					"LastUpdatedDate":      "2023-09-26T10:30:00Z",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region",
				"ARN", "Shards", "DestinationId", "RetentionPeriodHours",
				"EncryptionType", "Status", "CreatedDate", "LastUpdatedDate",
			},
			wantValues: []string{
				"Analytics", "Kinesis", "test-stream", "us-east-1",
				"arn:aws:kinesis:us-east-1:123456789012:stream/test-stream", "2", "", "24",
				"KMS", "ACTIVE", "2023-09-25T01:07:55Z", "2023-09-26T10:30:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &KinesisCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
