package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewIAMRoleCollector(t *testing.T) {
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

			collector, err := NewIAMRoleCollector(cfg, tt.regions, nameResolver)
			require.NoError(t, err)
			require.NotNil(t, collector)
			assert.NotNil(t, collector.client)
			assert.NotNil(t, collector.nameResolver)
		})
	}
}

func TestIAMRoleCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "iam_role", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &IAMRoleCollector{
				client: &iam.Client{},
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestIAMRoleCollector_GetColumns(t *testing.T) {
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
				SubCategory1: "IAM",
				Name:         "test-role",
				Region:       "Global",
				ARN:          "arn:aws:iam::123456789012:role/test-role",
				RawData: map[string]any{
					"Path":                "/",
					"AttachedPolicies":    "ReadOnlyAccess,PowerUserAccess",
					"PermissionsBoundary": "arn:aws:iam::123456789012:policy/boundary",
					"CreateDate":          "2023-09-25T01:07:55Z",
					"LastUsedDate":        "2023-09-26T10:30:00Z",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region",
				"ARN", "Path", "AttachedPolicies", "PermissionsBoundary", "CreateDate", "LastUsedDate",
			},
			wantValues: []string{
				"Security", "IAM", "test-role", "Global",
				"arn:aws:iam::123456789012:role/test-role", "/", "ReadOnlyAccess,PowerUserAccess", "arn:aws:iam::123456789012:policy/boundary", "2023-09-25T01:07:55Z", "2023-09-26T10:30:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &IAMRoleCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
