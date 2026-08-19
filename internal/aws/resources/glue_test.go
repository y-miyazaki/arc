package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewGlueCollector(t *testing.T) {
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

			collector, err := NewGlueCollector(cfg, tt.regions, nameResolver)
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

func TestGlueCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "glue", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &GlueCollector{
				clients: map[string]*glue.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestGlueCollector_GetColumns(t *testing.T) {
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
				SubCategory1: "Glue",
				Name:         "test-job",
				Region:       "us-east-1",
				ARN:          "test-job",
				RawData: map[string]any{
					"Description":     "Test Glue job",
					"RoleARN":         "arn:aws:iam::123456789012:role/GlueServiceRole",
					"Timeout":         "60",
					"WorkerType":      "G.1X",
					"NumberOfWorkers": "2",
					"MaxRetries":      "0",
					"GlueVersion":     "3.0",
					"Language":        "python",
					"ScriptLocation":  "s3://my-bucket/scripts/test.py",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region",
				"ID", "Description", "RoleARN", "Timeout", "WorkerType",
				"NumberOfWorkers", "MaxRetries", "GlueVersion", "Language", "ScriptLocation",
			},
			wantValues: []string{
				"Analytics", "Glue", "test-job", "us-east-1",
				"test-job", "Test Glue job", "arn:aws:iam::123456789012:role/GlueServiceRole", "60", "G.1X",
				"2", "0", "3.0", "python", "s3://my-bucket/scripts/test.py",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &GlueCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
