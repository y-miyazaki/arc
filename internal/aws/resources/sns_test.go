package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewSNSCollector(t *testing.T) {
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

			collector, err := NewSNSCollector(cfg, tt.regions, nameResolver)
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

func TestSNSCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "sns", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &SNSCollector{
				clients: make(map[string]*sns.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestSNSCollector_GetColumns(t *testing.T) {
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
				Category:     "sns",
				SubCategory1: "Topic",
				Name:         "test-topic",
				Region:       "us-east-1",
				ARN:          "arn:aws:sns:us-east-1:123456789012:test-topic",
				RawData: map[string]any{
					"DisplayName": "Test Topic",
					"Owner":       "123456789012",
					"Policy":      "{\"Version\":\"2012-10-17\"}",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"DisplayName", "Owner", "Policy",
			},
			wantValues: []string{
				"sns", "Topic", "test-topic", "us-east-1", "arn:aws:sns:us-east-1:123456789012:test-topic",
				"Test Topic", "123456789012", "{\"Version\":\"2012-10-17\"}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &SNSCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}

func TestSNSCollector_Collect_NoClient(t *testing.T) {
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
			collector := &SNSCollector{clients: map[string]*sns.Client{}}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestSNSCollector_Collect_ListTopicsError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &SNSCollector{
		clients: map[string]*sns.Client{
			"us-east-1": sns.NewFromConfig(cfg),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list SNS topics")
}
