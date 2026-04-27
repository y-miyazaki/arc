package resources

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/y-miyazaki/arc/internal/aws/helpers"
)

func TestNewStepFunctionsCollector(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}
	regions := []string{"us-east-1", "eu-west-1"}

	nameResolver, err := helpers.NewNameResolver(cfg, regions)
	require.NoError(t, err)

	collector, err := NewStepFunctionsCollector(cfg, regions, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Len(t, collector.clients, len(regions))
	assert.Contains(t, collector.clients, "us-east-1")
	assert.Contains(t, collector.clients, "eu-west-1")
	assert.NotNil(t, collector.nameResolver)
}

func TestNewStepFunctionsCollector_EmptyRegions(t *testing.T) {
	cfg := &aws.Config{
		Region: "us-east-1",
	}

	nameResolver, err := helpers.NewNameResolver(cfg, []string{})
	require.NoError(t, err)

	collector, err := NewStepFunctionsCollector(cfg, []string{}, nameResolver)

	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.Empty(t, collector.clients)
	assert.NotNil(t, collector.nameResolver)
}

func TestStepFunctionsCollector_Basic(t *testing.T) {
	collector := &StepFunctionsCollector{
		clients: make(map[string]*sfn.Client),
	}
	assert.Equal(t, "stepfunctions", collector.Name())
	assert.True(t, collector.ShouldSort())
}

func TestStepFunctionsCollector_Collect_NoClient(t *testing.T) {
	collector := &StepFunctionsCollector{
		clients: make(map[string]*sfn.Client),
	}

	_, err := collector.Collect(context.Background(), "us-west-2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no client found for region")
}

func TestStepFunctionsCollector_GetColumns(t *testing.T) {
	collector := &StepFunctionsCollector{}
	columns := collector.GetColumns()

	expectedHeaders := []string{
		"Category", "SubCategory1", "Name", "Region", "ARN", "Comment", "Type", "RoleARN",
		"LoggingLevel", "LoggingIncludeExecutionData", "LogDestination", "TracingEnabled",
		"EncryptionType", "KMSKeyID", "KMSDataKeyReusePeriodSeconds", "Definition",
		"RevisionID", "Status", "CreatedDate",
	}

	assert.Len(t, columns, len(expectedHeaders))
	for i, column := range columns {
		assert.Equal(t, expectedHeaders[i], column.Header)
	}

	sampleResource := Resource{
		Category:     "stepfunctions",
		SubCategory1: "StateMachine",
		Name:         "order-workflow",
		Region:       "us-east-1",
		ARN:          "arn:aws:states:us-east-1:123456789012:stateMachine:order-workflow",
		RawData: map[string]any{
			"Type":                         "STANDARD",
			"Status":                       "ACTIVE",
			"RoleARN":                      "arn:aws:iam::123456789012:role/step-functions-role",
			"LoggingLevel":                 "ALL",
			"LoggingIncludeExecutionData":  "true",
			"LogDestination":               "arn:aws:logs:us-east-1:123456789012:log-group:/aws/states/order:*",
			"TracingEnabled":               "true",
			"EncryptionType":               "CUSTOMER_MANAGED_KMS_KEY",
			"KMSKeyID":                     "arn:aws:kms:us-east-1:123456789012:key/test",
			"KMSDataKeyReusePeriodSeconds": "300",
			"Definition":                   `{"Comment":"Order processing workflow"}`,
			"RevisionID":                   "revision-1",
			"CreatedDate":                  "2026-03-17T00:00:00Z",
			"Comment":                      "Order processing workflow",
		},
	}

	expectedValues := []string{
		"stepfunctions", "StateMachine", "order-workflow", "us-east-1",
		"arn:aws:states:us-east-1:123456789012:stateMachine:order-workflow", "Order processing workflow", "STANDARD",
		"arn:aws:iam::123456789012:role/step-functions-role", "ALL", "true",
		"arn:aws:logs:us-east-1:123456789012:log-group:/aws/states/order:*", "true",
		"CUSTOMER_MANAGED_KMS_KEY", "arn:aws:kms:us-east-1:123456789012:key/test", "300",
		`{"Comment":"Order processing workflow"}`, "revision-1", "ACTIVE", "2026-03-17T00:00:00Z",
	}

	for i, column := range columns {
		assert.Equal(t, expectedValues[i], column.Value(sampleResource), "Column %d (%s) value mismatch", i, column.Header)
	}
}

func TestStepFunctionsCollector_Collect_ListStateMachinesError(t *testing.T) {
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: aws.AnonymousCredentials{},
	}
	collector := &StepFunctionsCollector{
		clients: map[string]*sfn.Client{
			"us-east-1": sfn.NewFromConfig(cfg),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collector.Collect(ctx, "us-east-1")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to list state machines")
}

func TestGetDefinitionComment(t *testing.T) {
	valid := `{"Comment":"workflow comment"}`
	invalid := "not-json"
	missing := `{"Name":"workflow"}`

	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{name: "nil definition", input: nil, expected: ""},
		{name: "invalid json", input: aws.String(invalid), expected: ""},
		{name: "missing comment", input: aws.String(missing), expected: ""},
		{name: "valid comment", input: aws.String(valid), expected: "workflow comment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getDefinitionComment(tt.input))
		})
	}
}

func TestStepFunctionsConfigHelpers(t *testing.T) {
	period := int32(300)
	encryption := &sfntypes.EncryptionConfiguration{
		KmsKeyId:                     aws.String("key-arn"),
		KmsDataKeyReusePeriodSeconds: &period,
		Type:                         sfntypes.EncryptionTypeCustomerManagedKmsKey,
	}

	logConfig := &sfntypes.LoggingConfiguration{
		Destinations: []sfntypes.LogDestination{{
			CloudWatchLogsLogGroup: &sfntypes.CloudWatchLogsLogGroup{LogGroupArn: aws.String("log-arn")},
		}},
		IncludeExecutionData: true,
		Level:                sfntypes.LogLevelAll,
	}

	tracing := &sfntypes.TracingConfiguration{Enabled: true}

	assert.Equal(t, "", getEncryptionKeyID(nil))
	assert.Equal(t, "key-arn", getEncryptionKeyID(encryption))

	assert.Equal(t, "", getEncryptionKeyReusePeriod(nil))
	assert.Equal(t, "", getEncryptionKeyReusePeriod(&sfntypes.EncryptionConfiguration{}))
	assert.Equal(t, int32(300), getEncryptionKeyReusePeriod(encryption))

	assert.Equal(t, "", getEncryptionType(nil))
	assert.Equal(t, string(sfntypes.EncryptionTypeCustomerManagedKmsKey), getEncryptionType(encryption))

	assert.Equal(t, "", getLogDestination(nil))
	assert.Equal(t, "", getLogDestination(&sfntypes.LoggingConfiguration{}))
	assert.Equal(t, "", getLogDestination(&sfntypes.LoggingConfiguration{Destinations: []sfntypes.LogDestination{{}}}))
	assert.Equal(t, "log-arn", getLogDestination(logConfig))

	assert.Equal(t, "", getLoggingIncludeExecutionData(nil))
	assert.Equal(t, true, getLoggingIncludeExecutionData(logConfig))

	assert.Equal(t, "", getLoggingLevel(nil))
	assert.Equal(t, string(sfntypes.LogLevelAll), getLoggingLevel(logConfig))

	assert.Equal(t, "", getTracingEnabled(nil))
	assert.Equal(t, true, getTracingEnabled(tracing))
}
