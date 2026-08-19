package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewSESCollector(t *testing.T) {
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

			collector, err := NewSESCollector(cfg, tt.regions, nameResolver)
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

func TestSESCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "ses", wantSort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &SESCollector{
				clients: make(map[string]*sesv2.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestSESCollector_Collect_NoClient(t *testing.T) {
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
			collector := &SESCollector{
				clients: make(map[string]*sesv2.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestSESCollector_GetColumns(t *testing.T) {
	collector := &SESCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ARN",
		"IdentityType", "VerificationStatus", "DkimStatus", "DkimTokens",
		"MailFromDomain", "MailFromDomainStatus", "BehaviorOnMXFailure", "DefaultConfigurationSet",
		"SendingEnabled", "ReputationMetricsEnabled", "TrackingOptions",
		"DestinationType", "DestinationARN", "EventTypes",
	}

	// Be tolerant: ensure each expected header exists somewhere in the columns
	actualHeaders := make([]string, 0, len(columns))
	for _, c := range columns {
		actualHeaders = append(actualHeaders, c.Header)
	}
	for _, h := range expectedHeaders {
		assert.Contains(t, actualHeaders, h, "expected header %s not found", h)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "email",
		SubCategory1: "SES",
		SubCategory2: "Identity",
		Name:         "test@example.com",
		Region:       "us-east-1",
		ARN:          "",
		RawData: map[string]any{
			"IdentityType":            "EmailAddress",
			"VerificationStatus":      "Verified",
			"DkimStatus":              "Enabled",
			"DkimTokens":              "[token1 token2]",
			"MailFromDomain":          "mail.example.com",
			"MailFromDomainStatus":    "Success",
			"BehaviorOnMXFailure":     "UseDefaultValue",
			"DefaultConfigurationSet": "",
		},
	}

	// Build a map header->value for assertions so we don't depend on column ordering
	valueByHeader := map[string]string{}
	for _, col := range columns {
		valueByHeader[col.Header] = col.Value(sampleResource)
	}

	expectedValues := map[string]string{
		"Category":                "email",
		"SubCategory1":            "SES",
		"SubCategory2":            "Identity",
		"Name":                    "test@example.com",
		"Region":                  "us-east-1",
		"ARN":                     "",
		"IdentityType":            "EmailAddress",
		"VerificationStatus":      "Verified",
		"DkimStatus":              "Enabled",
		"DkimTokens":              "[token1 token2]",
		"MailFromDomain":          "mail.example.com",
		"MailFromDomainStatus":    "Success",
		"BehaviorOnMXFailure":     "UseDefaultValue",
		"DefaultConfigurationSet": "",
	}

	for h, expected := range expectedValues {
		assert.Equal(t, expected, valueByHeader[h], "value mismatch for header %s", h)
	}

	// Test Value functions with ConfigurationSet sample resource
	configSetResource := Resource{
		Category:     "email",
		SubCategory1: "SES",
		SubCategory2: "ConfigurationSet",
		Name:         "test-config-set",
		Region:       "us-east-1",
		ARN:          "",
		RawData: map[string]any{
			"SendingEnabled":           "true",
			"ReputationMetricsEnabled": "true",
			"TrackingOptions":          "example.com",
		},
	}

	// Build header->value for config set
	cfgValues := map[string]string{}
	for _, col := range columns {
		cfgValues[col.Header] = col.Value(configSetResource)
	}

	configSetExpected := map[string]string{
		"Category":                 "email",
		"SubCategory1":             "SES",
		"SubCategory2":             "ConfigurationSet",
		"Name":                     "test-config-set",
		"Region":                   "us-east-1",
		"ARN":                      "",
		"SendingEnabled":           "true",
		"ReputationMetricsEnabled": "true",
		"TrackingOptions":          "example.com",
	}

	for h, expected := range configSetExpected {
		assert.Equal(t, expected, cfgValues[h], "ConfigSet value mismatch for header %s", h)
	}
}
