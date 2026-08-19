package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewDynamoDBCollector(t *testing.T) {
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

			collector, err := NewDynamoDBCollector(cfg, tt.regions, nameResolver)
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

func TestDynamoDBCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "dynamodb", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &DynamoDBCollector{
				clients: make(map[string]*dynamodb.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestDynamoDBCollector_Collect_NoClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		region       string
		wantContains string
	}{
		{name: "missing client returns error", region: "us-west-2", wantContains: "no client found for region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &DynamoDBCollector{
				clients: make(map[string]*dynamodb.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestDynamoDBCollector_GetColumns(t *testing.T) {
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
				Category:     "DynamoDB",
				SubCategory1: "Table",
				Name:         "test-table",
				Region:       "us-east-1",
				ARN:          "arn:aws:dynamodb:us-east-1:123456789012:table/test-table",
				RawData: map[string]any{
					"AttributeDefinitions":       "id:S",
					"BillingMode":                "PAY_PER_REQUEST",
					"StreamEnabled":              "false",
					"GlobalTable":                "false",
					"PointInTimeRecovery":        "ENABLED",
					"RecoveryPeriodInDays":       "35",
					"EarliestRestorableDateTime": "2023-01-01T00:00:00Z",
					"LatestRestorableDateTime":   "2023-12-01T00:00:00Z",
					"DeletionProtection":         "ENABLED",
					"TTLAttribute":               "ttl",
					"SSE":                        "ENABLED",
					"KmsKey":                     "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
					"ItemCount":                  "1000",
					"TableSize":                  "1048576",
					"Status":                     "ACTIVE",
					"CreationDateTime":           "2023-09-25T01:07:55Z",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"AttributeDefinitions", "BillingMode", "StreamEnabled", "GlobalTable", "PointInTimeRecovery",
				"RecoveryPeriodInDays", "EarliestRestorableDateTime", "LatestRestorableDateTime", "DeletionProtection",
				"TTLAttribute", "SSE", "KmsKey", "ItemCount", "TableSize(Bytes)", "Status", "CreationDateTime",
			},
			wantValues: []string{
				"DynamoDB", "Table", "test-table", "us-east-1", "arn:aws:dynamodb:us-east-1:123456789012:table/test-table",
				"id:S", "PAY_PER_REQUEST", "false", "false", "ENABLED",
				"35", "2023-01-01T00:00:00Z", "2023-12-01T00:00:00Z", "ENABLED",
				"ttl", "ENABLED", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "1000", "1048576", "ACTIVE", "2023-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &DynamoDBCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
