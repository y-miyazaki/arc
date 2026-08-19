package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewEventBridgeCollector(t *testing.T) {
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

			collector, err := NewEventBridgeCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.Len(t, collector.ebClients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.ebClients, region)
			}
			assert.Len(t, collector.schClients, tt.wantLen)
			for _, region := range tt.regions {
				assert.Contains(t, collector.schClients, region)
			}
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestEventBridgeCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "eventbridge", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &EventBridgeCollector{
				ebClients:  map[string]*eventbridge.Client{},
				schClients: map[string]*scheduler.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestEventBridgeCollector_GetColumns(t *testing.T) {
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
				Category:     "EventBridge",
				SubCategory1: "Rule",
				Name:         "test-rule",
				Region:       "us-east-1",
				ARN:          "arn:aws:events:us-east-1:123456789012:rule/test-rule",
				RawData: map[string]any{
					"Description":             "Test EventBridge rule",
					"RoleARN":                 "arn:aws:iam::123456789012:role/EventBridgeRole",
					"ScheduleExpression":      "rate(1 hour)",
					"Target":                  "arn:aws:lambda:us-east-1:123456789012:function:MyFunction",
					"RetryMaxAttempts":        "3",
					"RetryMaxEventAgeSeconds": "3600",
					"State":                   "ENABLED",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"Description", "RoleARN", "ScheduleExpression", "Target", "RetryMaxAttempts",
				"RetryMaxEventAgeSeconds", "State",
			},
			wantValues: []string{
				"EventBridge", "Rule", "test-rule", "us-east-1", "arn:aws:events:us-east-1:123456789012:rule/test-rule",
				"Test EventBridge rule", "arn:aws:iam::123456789012:role/EventBridgeRole", "rate(1 hour)", "arn:aws:lambda:us-east-1:123456789012:function:MyFunction", "3",
				"3600", "ENABLED",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &EventBridgeCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
