package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCognitoIdentityPoolCollector(t *testing.T) {
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

			collector, err := NewCognitoIdentityPoolCollector(cfg, tt.regions, nameResolver)
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

func TestCognitoIdentityPoolCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "cognito_identity_pool", wantSort: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &CognitoIdentityPoolCollector{
				clients: make(map[string]*cognitoidentity.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestCognitoIdentityPoolCollector_Collect_NoClient(t *testing.T) {
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
			collector := &CognitoIdentityPoolCollector{
				clients: make(map[string]*cognitoidentity.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestCognitoIdentityPoolCollector_GetColumns(t *testing.T) {
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
				Category:     "Cognito",
				SubCategory1: "IdentityPool",
				SubCategory2: "",
				Name:         "test-identity-pool",
				Region:       "us-east-1",
				ARN:          "us-east-1:12345678-1234-1234-1234-123456789012",
				RawData: map[string]any{
					"AllowUnauthenticated":      "true",
					"DeveloperProviderName":     "dev-provider",
					"SupportedLoginProviders":   []string{"graph.facebook.com=12345"},
					"CognitoIdentityProviders":  []string{"cognito-idp.us-east-1.amazonaws.com/region=clientid"},
					"OpenIdConnectProviderARNs": []string{"arn:aws:..."},
					"SamlProviderARNs":          []string{"arn:aws:saml:..."},
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ID",
				"AllowUnauthenticated", "DeveloperProviderName", "SupportedLoginProviders",
				"CognitoIdentityProviders", "OpenIdConnectProviderARNs", "SamlProviderARNs",
			},
			wantValues: []string{
				"Cognito", "IdentityPool", "", "test-identity-pool", "us-east-1", "us-east-1:12345678-1234-1234-1234-123456789012",
				"true", "dev-provider", "graph.facebook.com=12345",
				"cognito-idp.us-east-1.amazonaws.com/region=clientid", "arn:aws:...", "arn:aws:saml:...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &CognitoIdentityPoolCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}

func TestCognitoIdentityPoolCollector_Collect_ListIdentityPoolsError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &CognitoIdentityPoolCollector{
		clients: map[string]*cognitoidentity.Client{
			"us-east-1": cognitoidentity.NewFromConfig(cfg),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to collect identity pools")
	assert.ErrorContains(t, err, "failed to list identity pools")
}

func TestCollectIdentityPools_ListIdentityPoolsError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	client := cognitoidentity.NewFromConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collectIdentityPools(ctx, "us-east-1", client)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list identity pools")
}
