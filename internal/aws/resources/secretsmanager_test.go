package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewSecretsManagerCollector(t *testing.T) {
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

			collector, err := NewSecretsManagerCollector(cfg, tt.regions, nameResolver)
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

func TestSecretsManagerCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "secretsmanager", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &SecretsManagerCollector{
				clients: make(map[string]*secretsmanager.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestSecretsManagerCollector_GetColumns(t *testing.T) {
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
				Category:     "Security",
				SubCategory1: "SecretsManager",
				Name:         "test-secret",
				Region:       "us-east-1",
				ARN:          "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret-AbCdEf",
				RawData: map[string]any{
					"Description":       "Test secret",
					"KmsKey":            "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
					"RotationEnabled":   "true",
					"RotationLambdaARN": "arn:aws:lambda:us-east-1:123456789012:function:rotation-function",
					"SecretString":      "{\n  \"username\": \"admin\"\n}",
					"LastAccessedDate":  "2023-09-24T01:07:55Z",
					"LastRotatedDate":   "2023-09-25T01:07:55Z",
					"LastChangedDate":   "2023-09-26T01:07:55Z",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"Description", "KmsKey", "RotationEnabled", "RotationLambdaARN", "SecretString", "LastAccessedDate", "LastRotatedDate", "LastChangedDate",
			},
			wantValues: []string{
				"Security", "SecretsManager", "test-secret", "us-east-1", "arn:aws:secretsmanager:us-east-1:123456789012:secret:test-secret-AbCdEf",
				"Test secret", "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", "true", "arn:aws:lambda:us-east-1:123456789012:function:rotation-function", "{\n  \"username\": \"admin\"\n}", "2023-09-24T01:07:55Z", "2023-09-25T01:07:55Z", "2023-09-26T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &SecretsManagerCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
