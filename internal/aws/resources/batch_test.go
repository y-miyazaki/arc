package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewBatchCollector(t *testing.T) {
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

			collector, err := NewBatchCollector(cfg, tt.regions, nameResolver)
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

func TestBatchCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "batch", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &BatchCollector{
				clients: make(map[string]*batch.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestBatchCollector_Collect_NoClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		region       string
		wantContains string
	}{
		{name: "missing client returns error", region: "us-west-2", wantContains: "no Batch client found for region"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &BatchCollector{
				clients: make(map[string]*batch.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestBatchCollector_GetColumns(t *testing.T) {
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
				Category:     "Batch",
				SubCategory1: "Job Queue",
				Name:         "test-queue",
				Region:       "us-east-1",
				ARN:          "arn:aws:batch:us-east-1:123456789012:job-queue/test-queue",
				RawData: map[string]any{
					"Priority":              "1",
					"Type":                  "EC2",
					"JobRoleArn":            "arn:aws:iam::123456789012:role/BatchJobRole",
					"ExecutionRoleArn":      "arn:aws:iam::123456789012:role/BatchExecutionRole",
					"Image":                 "busybox",
					"vCPU":                  "1",
					"Memory":                "512",
					"CpuArchitecture":       "X86_64",
					"OperatingSystemFamily": "LINUX",
					"Timeout":               "3600",
					"JSON":                  "{}",
					"Status":                "ACTIVE",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"Priority", "Type", "JobRoleArn", "ExecutionRoleArn", "Image",
				"vCPU", "Memory", "CpuArchitecture", "OperatingSystemFamily", "Timeout", "JSON", "Status",
			},
			wantValues: []string{
				"Batch", "Job Queue", "test-queue", "us-east-1", "arn:aws:batch:us-east-1:123456789012:job-queue/test-queue",
				"1", "EC2", "arn:aws:iam::123456789012:role/BatchJobRole", "arn:aws:iam::123456789012:role/BatchExecutionRole", "busybox",
				"1", "512", "X86_64", "LINUX", "3600", "{}", "ACTIVE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &BatchCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
