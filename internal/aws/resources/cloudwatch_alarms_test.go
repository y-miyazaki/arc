package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCloudWatchAlarmsCollector(t *testing.T) {
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

			collector, err := NewCloudWatchAlarmsCollector(cfg, tt.regions, nameResolver)
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

func TestCloudWatchAlarmsCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "cloudwatch_alarms", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &CloudWatchAlarmsCollector{
				clients: make(map[string]*cloudwatch.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestCloudWatchAlarmsCollector_Collect_NoClient(t *testing.T) {
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
			collector := &CloudWatchAlarmsCollector{
				clients: make(map[string]*cloudwatch.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestCloudWatchAlarmsCollector_GetColumns(t *testing.T) {
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
				Category:     "CloudWatch",
				SubCategory1: "Alarms",
				Name:         "test-alarm",
				Region:       "us-east-1",
				ARN:          "arn:aws:cloudwatch:us-east-1:123456789012:alarm:test-alarm",
				RawData: map[string]any{
					"MetricName":         "CPUUtilization",
					"Namespace":          "AWS/EC2",
					"Statistic":          "Average",
					"Threshold":          "80.0",
					"ComparisonOperator": "GreaterThanThreshold",
					"EvaluationPeriods":  "2",
					"Period":             "300",
					"TreatMissingData":   "missing",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"MetricName", "Namespace", "Statistic", "Threshold", "ComparisonOperator", "EvaluationPeriods", "Period", "TreatMissingData",
			},
			wantValues: []string{
				"CloudWatch", "Alarms", "test-alarm", "us-east-1", "arn:aws:cloudwatch:us-east-1:123456789012:alarm:test-alarm",
				"CPUUtilization", "AWS/EC2", "Average", "80.0", "GreaterThanThreshold", "2", "300", "missing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &CloudWatchAlarmsCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
