package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewECRCollector(t *testing.T) {
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

			collector, err := NewECRCollector(cfg, tt.regions, nameResolver)
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

func TestECRCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "ecr", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &ECRCollector{
				clients: map[string]*ecr.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestECRCollector_GetColumns(t *testing.T) {
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
				Category:     "ECR",
				SubCategory1: "Repository",
				Name:         "test-repo",
				Region:       "us-east-1",
				RawData: map[string]any{
					"URI":             "123456789012.dkr.ecr.us-east-1.amazonaws.com/test-repo",
					"Mutability":      "MUTABLE",
					"Encryption":      "KMS",
					"KMSKey":          "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
					"ScanOnPush":      "true",
					"LifecyclePolicy": "Yes",
					"ImageCount":      "5",
					"CreatedAt":       "2023-09-25T01:07:55Z",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region",
				"URI", "Mutability", "Encryption", "KMSKey", "ScanOnPush",
				"LifecyclePolicy", "ImageCount", "CreatedAt",
			},
			wantValues: []string{
				"ECR", "Repository", "test-repo", "us-east-1",
				"123456789012.dkr.ecr.us-east-1.amazonaws.com/test-repo", "MUTABLE", "KMS", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "true",
				"Yes", "5", "2023-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &ECRCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
