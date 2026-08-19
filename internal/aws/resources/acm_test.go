package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

// MockACMClient is a testify/mock-based mock for ACM client.
type MockACMClient struct {
	mock.Mock
}

func (m *MockACMClient) ListCertificates(ctx context.Context, params *acm.ListCertificatesInput, optFns ...func(*acm.Options)) (*acm.ListCertificatesOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*acm.ListCertificatesOutput), args.Error(1)
}

func (m *MockACMClient) DescribeCertificate(ctx context.Context, params *acm.DescribeCertificateInput, optFns ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*acm.DescribeCertificateOutput), args.Error(1)
}

func TestNewACMCollector(t *testing.T) {
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

			collector, err := NewACMCollector(cfg, tt.regions, nameResolver)
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

func TestACMCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "acm", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &ACMCollector{
				clients: make(map[string]*acm.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestACMCollector_Collect_NoClient(t *testing.T) {
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
			collector := &ACMCollector{
				clients: make(map[string]*acm.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestACMCollector_GetColumns(t *testing.T) {
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
				SubCategory1: "ACM",
				Name:         "example.com",
				Region:       "us-east-1",
				ARN:          "arn:aws:acm:us-east-1:123456789012:certificate/test-cert",
				RawData: map[string]any{
					"Status":         "ISSUED",
					"Type":           "AMAZON_ISSUED",
					"KeyAlgorithm":   "RSA_2048",
					"InUse":          "test-alb",
					"RequestDate":    "2023-09-25T01:07:55Z",
					"IssuedDate":     "2023-09-25T01:07:55Z",
					"ExpirationDate": "2024-09-25T01:07:55Z",
					"CreatedDate":    "2023-09-25T01:07:55Z",
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "Name", "Region", "ARN",
				"Type", "KeyAlgorithm", "InUse", "Status", "CreatedDate", "IssuedDate", "ExpirationDate",
			},
			wantValues: []string{
				"Security", "ACM", "example.com", "us-east-1", "arn:aws:acm:us-east-1:123456789012:certificate/test-cert",
				"AMAZON_ISSUED", "RSA_2048", "test-alb", "ISSUED", "2023-09-25T01:07:55Z", "2023-09-25T01:07:55Z", "2024-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &ACMCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}
