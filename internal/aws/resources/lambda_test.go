package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewLambdaCollector(t *testing.T) {
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

			collector, err := NewLambdaCollector(cfg, tt.regions, nameResolver)
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

func TestLambdaCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "lambda", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &LambdaCollector{
				clients: map[string]*lambda.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestLambdaCollector_GetColumns(t *testing.T) {
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
				Category:     "Compute",
				SubCategory1: "Lambda",
				Name:         "test-function",
				Region:       "us-east-1",
				ARN:          "arn:aws:lambda:us-east-1:123456789012:function:test-function",
				RawData: map[string]any{
					"RoleARN":      "arn:aws:iam::123456789012:role/lambda-role",
					"Type":         "Function",
					"Runtime":      "python3.9",
					"Architecture": "x86_64",
					"MemorySize":   "128",
					"Timeout":      "30",
					"EnvVars":      "KEY1=value1,KEY2=value2",
					"LastModified": "2023-09-25T01:07:55.000+0000",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region",
				"ARN", "RoleARN", "Type", "Runtime", "Architecture",
				"MemorySize", "Timeout", "EnvVars", "LastModified",
			},
			wantValues: []string{
				"Compute", "Lambda", "test-function", "us-east-1",
				"arn:aws:lambda:us-east-1:123456789012:function:test-function", "arn:aws:iam::123456789012:role/lambda-role", "Function", "python3.9", "x86_64",
				"128", "30", "KEY1=value1,KEY2=value2", "2023-09-25T01:07:55.000+0000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &LambdaCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
