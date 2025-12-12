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
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	// Create a NameResolver for testing
	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewCognitoUserPoolCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, 2)
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewCognitoUserPoolCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	// Create a NameResolver even with empty regions
	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewCognitoUserPoolCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestCognitoUserPoolCollector_Basic(t *testing.T) {
	collector := &CognitoUserPoolCollector{
		clients: make(map[string]*cognitoidentityprovider.Client),
	}
	assert.Equal(t, "cognito_user_pool", collector.Name())
	assert.False(t, collector.ShouldSort())
}

func TestCognitoUserPoolCollector_Collect_NoClient(t *testing.T) {
	collector := &CognitoUserPoolCollector{
		clients: make(map[string]*cognitoidentityprovider.Client),
	}

	ctx := context.Background()
	_, err := collector.Collect(ctx, "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestCognitoUserPoolCollector_GetColumns(t *testing.T) {
	collector := &CognitoUserPoolCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "SubCategory2", "Name", "Region", "ARN", "ID", "Description",
		"MfaConfiguration", "AliasAttributes", "UsernameAttributes", "AutoVerifiedAttributes",
		"PasswordPolicy", "LambdaConfig", "Precedence", "RoleArn", "AttachedUsers", "Groups", "Attributes",
		"CreationDate", "LastModifiedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	// Test Value functions with sample resource
	sampleResource := Resource{
		Category:     "Cognito",
		SubCategory1: "UserPool",
		SubCategory2: "",
		Name:         "test-user-pool",
		Region:       "us-east-1",
		ARN:          "arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_123456789",
		RawData: map[string]interface{}{
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
	}

	expectedValues := []string{
		"Cognito", "UserPool", "", "test-user-pool", "us-east-1",
		"arn:aws:cognito-idp:us-east-1:123456789012:userpool/us-east-1_123456789", "us-east-1_123456789", "Test description",
		"OFF", "email", "email", "email", "MinimumLength=8\nRequireNumbers=true", "PreSignUp=arn:aws:lambda:...",
		"10", "arn:aws:iam::123456789012:role/test", "user1", "group1",
		"AccountEnabled=true\nemail=test@example.com\nUserStatus=CONFIRMED\nVerifiedEmail=true\nVerifiedPhone=false",
		"2023-09-25T01:07:55Z", "2023-09-25T01:07:55Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}
