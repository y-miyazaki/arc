package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewCognitoUserPoolCollector(t *testing.T) {
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

			collector, err := NewCognitoUserPoolCollector(cfg, tt.regions, nameResolver)
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

func TestCognitoUserPoolCollector_Basic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantName string
		wantSort bool
	}{
		{name: "reports name and sort", wantName: "cognito_user_pool", wantSort: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &CognitoUserPoolCollector{
				clients: make(map[string]*cognitoidentityprovider.Client),
			}
			assert.Equal(t, tt.wantName, collector.Name())
			assert.Equal(t, tt.wantSort, collector.ShouldSort())
		})
	}
}

func TestCognitoUserPoolCollector_Collect_NoClient(t *testing.T) {
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
			collector := &CognitoUserPoolCollector{
				clients: make(map[string]*cognitoidentityprovider.Client),
			}
			_, err := collector.Collect(context.Background(), tt.region)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantContains)
		})
	}
}

func TestCognitoUserPoolCollector_GetColumns(t *testing.T) {
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
				SubCategory1: "UserPool",
				SubCategory2: "",
				Name:         "test-user-pool",
				Region:       "us-east-1",
				ARN:          "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_123456789",
				RawData: map[string]any{
					"ID":                     "us-east-1_123456789",
					"CreationDate":           "2023-09-25T01:07:55Z",
					"LastModifiedDate":       "2023-09-25T01:07:55Z",
					"Description":            "Test description",
					"MfaConfiguration":       "OFF",
					"AliasAttributes":        []string{"email"},
					"UsernameAttributes":     []string{"email"},
					"AutoVerifiedAttributes": []string{"email"},
					"PasswordPolicy":         []string{"MinimumLength=8", "RequireNumbers=true"},
					"LambdaConfig":           []string{"PreSignUp=arn:aws:lambda:..."},
					"Precedence":             "10",
					"Groups":                 []string{"group1"},
					"Attributes":             []string{"email=test@example.com", "AccountEnabled=true", "UserStatus=CONFIRMED", "VerifiedEmail=true", "VerifiedPhone=false"},
					"RoleArn":                "arn:aws:iam::123456789012:role/test",
					"AttachedUsers":          []string{"user1"},
				},
			},
			wantHeaders: []string{
				"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ARN", "ID", "Description",
				"MfaConfiguration", "AliasAttributes", "UsernameAttributes", "AutoVerifiedAttributes",
				"PasswordPolicy", "LambdaConfig", "Precedence", "RoleArn", "AttachedUsers", "Groups", "Attributes",
				"CreationDate", "LastModifiedDate",
			},
			wantValues: []string{
				"Cognito", "UserPool", "", "test-user-pool", "us-east-1",
				"arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_123456789", "us-east-1_123456789", "Test description",
				"OFF", "email", "email", "email", "MinimumLength=8\nRequireNumbers=true", "PreSignUp=arn:aws:lambda:...",
				"10", "arn:aws:iam::123456789012:role/test", "user1", "group1",
				"AccountEnabled=true\nemail=test@example.com\nUserStatus=CONFIRMED\nVerifiedEmail=true\nVerifiedPhone=false",
				"2023-09-25T01:07:55Z", "2023-09-25T01:07:55Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			collector := &CognitoUserPoolCollector{}
			columns := collector.GetColumns()
			require.Len(t, columns, len(tt.wantHeaders))
			for i, column := range columns {
				assert.Equal(t, tt.wantHeaders[i], column.Header)
				assert.Equal(t, tt.wantValues[i], column.Value(tt.resource), "Column %d (%s) value mismatch", i, column.Header)
			}
		})
	}
}

func TestCognitoUserPoolCollector_Collect_ListUserPoolsError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	collector := &CognitoUserPoolCollector{
		clients: map[string]*cognitoidentityprovider.Client{
			"us-east-1": cognitoidentityprovider.NewFromConfig(cfg),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to collect user pools")
	assert.ErrorContains(t, err, "failed to list user pools")
}

func TestCollectUserPools_ListUserPoolsError(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1", Credentials: aws.AnonymousCredentials{}}
	client := cognitoidentityprovider.NewFromConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collectUserPools(ctx, "us-east-1", client)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list user pools")
}
